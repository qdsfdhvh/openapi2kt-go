package schema

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

type mergeRoot struct {
	d         Declaration
	pkg, full string
	priority  int
	key       string
	height    int
	deps      map[*mergeRoot]bool
	parents   []*mergeRoot
	remaining int
}

// MergeIdenticalModels compares validated owned IR, never expanded source trees.
// Mutual cycles and over-height dependency graphs retain nominal identity;
// direct owned self edges normalize inside the bounded declaration view.
func MergeIdenticalModels(plan *Plan, pkg string, components map[Declaration]bool, excluded func(string) bool) {
	roots := []*mergeRoot{}
	index := map[string]*mergeRoot{}
	aliases := map[string]string{}
	inventory := [][]Declaration{plan.Models}
	if plan.Client != nil {
		inventory = append(inventory, plan.Client.Models)
	}
	for i, list := range inventory {
		modelPkg := pkg + ".models"
		if i == 1 {
			modelPkg = pkg + ".api.models"
		}
		for _, d := range list {
			full := modelPkg + "." + d.Name()
			if a, ok := d.(*ScalarAlias); ok && plan.ownedAliases[a] != "" {
				aliases[full] = plan.ownedAliases[a]
			}
			if !dedupEligible(d) {
				continue
			}
			priority := 1
			if components[d] {
				priority = 0
			}
			if i == 1 {
				priority = 2
			}
			r := &mergeRoot{d: d, pkg: modelPkg, full: full, priority: priority, deps: map[*mergeRoot]bool{}}
			roots = append(roots, r)
			index[full] = r
		}
	}
	protected := mergeOpaqueOwners(plan, roots)
	for _, r := range roots {
		if excluded(r.full) {
			protected[r.full] = true
		}
		if protected[r.full] {
			delete(index, r.full)
		}
	}
	resolve := func(name string) (*mergeRoot, string) {
		for steps := 0; steps < MaxSchemaDepth; steps++ {
			if target := aliases[name]; target != "" {
				name = target
				continue
			}
			for prefix := name; prefix != ""; {
				if r := index[prefix]; r != nil {
					return r, strings.TrimPrefix(name, prefix)
				}
				i := strings.LastIndexByte(prefix, '.')
				if i < 0 {
					break
				}
				prefix = prefix[:i]
			}
			break
		}
		return nil, ""
	}
	// Dependency construction is bounded by the existing owned declaration tree.
	for _, r := range roots {
		if protected[r.full] {
			continue
		}
		names := map[string]string{}
		dedupOwnedNames(r.d, r.d.Name(), "$root", r.pkg, names)
		mergeTypes(r.d, func(t TypeRef) TypeRef {
			// Owned root/self and nested classifiers normalize to $root in
			// dedupView; hashing never expands their recursive edges.
			if t.Kind == Named && names[t.Name] == "" {
				if child, _ := resolve(t.Name); child != nil {
					r.deps[child] = true
				}
			}
			return t
		})
		for child := range r.deps {
			child.parents = append(child.parents, r)
		}
	}
	queue := []*mergeRoot{}
	for _, r := range roots {
		r.remaining = len(r.deps)
		if !protected[r.full] && r.remaining == 0 {
			queue = append(queue, r)
		}
	}
	for i := 0; i < len(queue); i++ {
		r := queue[i]
		r.height = 1
		for child := range r.deps {
			if child.height+1 > r.height {
				r.height = child.height + 1
			}
		}
		if r.height <= MaxSchemaDepth {
			names := map[string]string{}
			dedupOwnedNames(r.d, r.d.Name(), "$root", r.pkg, names)
			view := dedupView(r.d, names, true)
			mergeTypes(view, func(t TypeRef) TypeRef {
				if t.Kind == Named && !strings.HasPrefix(t.Name, "$root") {
					if child, tail := resolve(t.Name); child != nil && child.key != "" {
						t.Name = "$resolved:" + child.key + tail
					}
				}
				return t
			})
			data, err := json.Marshal(view)
			if err != nil {
				panic("invalid merge semantic view")
			}
			r.key = fmt.Sprintf("%T:%x", r.d, sha256.Sum256(data))
		}
		for _, parent := range r.parents {
			parent.remaining--
			if parent.remaining == 0 {
				queue = append(queue, parent)
			}
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].priority != roots[j].priority {
			return roots[i].priority < roots[j].priority
		}
		return roots[i].full < roots[j].full
	})
	canonical := map[string]*mergeRoot{}
	replacement := map[Declaration]Declaration{}
	rebases := map[string]string{}
	for _, r := range roots {
		if r.key == "" {
			continue
		}
		if previous := canonical[r.key]; previous != nil {
			plan.EmissionHints.transferEquivalent(previous.d, r.d)
			replacement[r.d] = plan.ownedAlias(r.d.Name(), previous.full)
			names := map[string]string{}
			dedupOwnedNames(r.d, r.d.Name(), previous.full, r.pkg, names)
			for name, target := range names {
				if strings.HasPrefix(name, r.pkg+".") {
					rebases[name] = target
				}
			}
		} else {
			canonical[r.key] = r
		}
	}
	// Only typed Named edges are rebound. Opaque incoming expressions protected
	// their owners; source accounting and original check pointers are untouched.
	mergeTypes(plan, func(t TypeRef) TypeRef {
		if t.Kind == Named {
			for steps := 0; steps < MaxSchemaDepth; steps++ {
				if name := aliases[t.Name]; name != "" {
					t.Name = name
					continue
				}
				if name := rebases[t.Name]; name != "" {
					t.Name = name
				}
				break
			}
		}
		return t
	})
	for i, d := range plan.Models {
		if alias := replacement[d]; alias != nil {
			plan.Models[i] = alias
		}
	}
	if plan.Client != nil {
		for i, d := range plan.Client.Models {
			if alias := replacement[d]; alias != nil {
				plan.Client.Models[i] = alias
			}
		}
		// Structural aliases can collapse formerly distinct nominal overloads.
		// Reuse natural-name reservations and path/method suffix allocation;
		// public model aliases/owners and name-lock occurrences stay unchanged.
		for i := range plan.Client.Groups {
			renameMethods(&plan.Client.Groups[i])
		}
	}
}

// walkTypedMerge visits typed private IR once per physical pointer. It does not
// interpret strings, JSON, maps, environment or filesystem state.
func walkTypedMerge(value any, visit func(reflect.Value) bool) {
	seen := map[uintptr]bool{}
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() || seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
			walk(v.Elem())
			return
		}
		if v.CanInterface() && visit(v) {
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Field(i).CanInterface() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(value))
}
func mergeTypes(value any, transform func(TypeRef) TypeRef) {
	walkTypedMerge(value, func(v reflect.Value) bool {
		if t, ok := v.Interface().(TypeRef); ok {
			var apply func(TypeRef) TypeRef
			apply = func(t TypeRef) TypeRef {
				t = transform(t)
				if t.Element != nil {
					element := apply(*t.Element)
					t.Element = &element
				}
				return t
			}
			if v.CanSet() {
				v.Set(reflect.ValueOf(apply(t)))
			} else {
				_ = apply(t)
			}
			return true
		}
		return false
	})
}
func mergeOpaqueOwners(plan *Plan, roots []*mergeRoot) map[string]bool {
	protected := map[string]bool{}
	byToken := map[string][]string{}
	for _, r := range roots {
		byToken[r.d.Name()] = append(byToken[r.d.Name()], r.full)
	}
	add := func(code string) {
		for _, token := range strings.FieldsFunc(code, func(r rune) bool { return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			for _, full := range byToken[token] {
				protected[full] = true
			}
		}
	}
	walkTypedMerge(plan, func(v reflect.Value) bool {
		switch item := v.Interface().(type) {
		case TypeRef:
			add(item.Serializer)
			if item.Kind == Scalar || item.Kind == InlineValue {
				add(item.Name)
			}
		case Constant:
			add(item.Initializer)
		case ScalarAlias:
			// A generated optimizer alias is a typed root edge; provider spellings
			// and other opaque alias targets cannot be safely rebound.
			if !v.CanAddr() || plan.ownedAliases[v.Addr().Interface().(*ScalarAlias)] == "" {
				add(item.Scalar)
			}
		}
		return false
	})
	return protected
}
