package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// A dictionary wrapper has one allOf reference and only neutral documentation
// siblings. It borrows the dictionary backing and value peers; any default
// factory belongs to the component that declares it.
type dictionaryAllOfSource struct {
	Owner                   *component
	Schema, Members, Member *document.Value
	Reference               *document.Value
	Pointer, Text           string
	MemberIndex             int
	MemberItems             []*document.Value
	MemberFields            []document.Member
	Canonical               *component
	Bindings                []dictionaryAllOfBinding
}

type dictionaryAllOfBinding struct {
	Owner                     *component
	Schema, Reference         *document.Value
	Target                    *component
	Nullable                  *bool
	Declaration               *ValueAlias
	Types                     []namedCollectionSourceType
	Factory                   string
	Fields                    []document.Member
	Nodes                     []*document.Value
	NullableValue             bool
	Wire, Name, Pointer, Text string
	Bytes                     string
	Height                    int
}

type dictionaryAllOfBudget struct {
	Work, Bytes int
}

func (b *dictionaryAllOfBudget) charge(work, size int, pointer string) *diagnostic.Diagnostic {
	if work > 16384-b.Work || size > (8<<20)-b.Bytes {
		return diagnostic.At("E_MODEL_LIMIT", "Dictionary wrapper source capture exceeds the cumulative work or byte budget.", pointer)
	}
	b.Work += work
	b.Bytes += size
	return nil
}

// Discovery grants only wrappers whose existing named reference chain ends
// at a direct dictionary. It does not turn ordinary object allOf roots into
// aliases or admit internal references, inherited defaults or non-neutral
// composition.
func (a *analyzer) prepareDictionaryAllOfSources() {
	type probe struct {
		source *dictionaryAllOfSource
		target *component
	}
	probes := map[*component]probe{}
	for _, c := range a.components {
		v := c.Schema
		members := get(v, "allOf")
		if c.Reference != "" || members.Kind() != document.Array {
			continue
		}
		view, err := parseReferenceWrapper(v, c.Pointer, true)
		if err != nil || view == nil {
			continue
		}
		index, eligible := dictionaryAllOfReferenceIndex(members.Items())
		if !eligible {
			continue
		}
		target, err := a.resolveDeclared(view.Reference, view.Pointer+"/$ref")
		if err != nil {
			continue
		}
		member := members.Items()[index]
		probes[c] = probe{
			source: &dictionaryAllOfSource{
				Owner: c, Schema: v, Members: members, Member: member,
				Reference: get(member, "$ref"), Pointer: view.Pointer + "/$ref",
				Text: view.Reference, MemberIndex: index, MemberItems: slices.Clone(members.Items()),
				MemberFields: slices.Clone(member.Members()),
			},
			target: target,
		}
	}
	// Memoized traversal handles forward references and bounds discovery work
	// independently of component declaration order. Cycles gain no admission.
	state := map[*component]uint8{}
	eligible := map[*component]bool{}
	for root := range probes {
		var path []*component
		ok := false
		for c := root; ; {
			if state[c] == 1 {
				break
			}
			if state[c] == 2 {
				ok = eligible[c]
				break
			}
			state[c] = 1
			path = append(path, c)
			if p := probes[c]; p.source != nil {
				c = p.target
				continue
			}
			if c.Reference != "" {
				target, err := a.resolveDeclared(c.Reference, c.Pointer+"/$ref")
				if err != nil {
					break
				}
				c = target
				continue
			}
			ok = dictionarySchema(c.Schema) && c.Scalar == "" && !c.ValueClass
			break
		}
		for _, c := range path {
			state[c], eligible[c] = 2, ok
		}
	}
	a.dictionaryAllOfSources = map[*component]*dictionaryAllOfSource{}
	for c, p := range probes {
		if !eligible[c] {
			continue
		}
		c.Reference, c.DictionaryAllOf = p.source.Text, p.source
		c.NullableDTO = false
		a.dictionaryAllOfSources[c] = p.source
	}
}

// Empty and documentation-only members add no value or nullability constraint.
// Other metadata and explicit nullable siblings retain their separate roles.
func dictionaryAllOfReferenceIndex(members []*document.Value) (int, bool) {
	index := -1
	for i, member := range members {
		if member.Kind() != document.Object {
			return 0, false
		}
		if has(member, "$ref") {
			if index >= 0 {
				return 0, false
			}
			index = i
			continue
		}
		for _, field := range member.Members() {
			if field.Name != "title" && field.Name != "description" && field.Name != "$comment" {
				return 0, false
			}
		}
	}
	return index, index >= 0
}

func dictionaryAllOfMembersMatch(source *dictionaryAllOfSource) bool {
	return get(source.Schema, "allOf") == source.Members &&
		slices.Equal(source.Members.Items(), source.MemberItems) &&
		source.MemberIndex >= 0 && source.MemberIndex < len(source.MemberItems) &&
		source.MemberItems[source.MemberIndex] == source.Member &&
		get(source.Member, "$ref") == source.Reference &&
		slices.Equal(source.Member.Members(), source.MemberFields)
}

func (a *analyzer) orderedDictionaryAllOfSources() []*dictionaryAllOfSource {
	sources := make([]*dictionaryAllOfSource, 0, len(a.dictionaryAllOfSources))
	for _, source := range a.dictionaryAllOfSources {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Pointer < sources[j].Pointer })
	return sources
}

// Freeze the actual original chain after the ordinary catalog binding pass.
// Canonical dictionary authority is also checked by the shared collection proof.
func (a *analyzer) captureDictionaryAllOfSources() *diagnostic.Diagnostic {
	rawByComponent := map[*component]string{}
	nodesByComponent := map[*component][]*document.Value{}
	budget := dictionaryAllOfBudget{}
	for _, source := range a.orderedDictionaryAllOfSources() {
		root := source.Owner
		source.Canonical = root.BindingTarget
		if !namedCollectionCandidate(source.Canonical) || !dictionarySchema(source.Canonical.Schema) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		seen := map[*component]bool{}
		for c := root; ; {
			if c == nil || seen[c] || len(seen) > MaxSchemaDepth {
				return schemaReferenceUnsupported(source.Pointer)
			}
			seen[c] = true
			raw, ok := rawByComponent[c]
			if !ok {
				data, _ := c.Schema.MarshalJSON()
				raw = string(data)
				rawByComponent[c] = raw
			}
			if err := budget.charge(1, len(raw), source.Pointer); err != nil {
				return err
			}
			if nodes, captured := nodesByComponent[c]; captured {
				if err := budget.charge(len(nodes), 0, source.Pointer); err != nil {
					return err
				}
			} else {
				nodes, err := captureDictionaryAllOfNodes(c.Schema, &budget, source.Pointer)
				if err != nil {
					return err
				}
				nodesByComponent[c] = nodes
			}
			binding := dictionaryAllOfBinding{
				Owner: c, Schema: c.Schema, Target: c.BindingTarget,
				Nullable: c.BindingNullable, Wire: c.Wire, Name: c.Name,
				Pointer: c.Pointer, Text: c.Reference, Bytes: raw, Height: c.AliasHeight,
				Fields: slices.Clone(c.Schema.Members()), Nodes: nodesByComponent[c],
			}
			if c.BindingNullable != nil {
				binding.NullableValue = *c.BindingNullable
			}
			if c.DictionaryAllOf != nil {
				binding.Reference = c.DictionaryAllOf.Reference
			} else {
				binding.Reference = get(c.Schema, "$ref")
			}
			source.Bindings = append(source.Bindings, binding)
			if c == source.Canonical {
				break
			}
			if c.Reference == "" || c.BindingTarget != source.Canonical {
				return schemaReferenceUnsupported(source.Pointer)
			}
			next, err := a.resolveDeclared(c.Reference, source.Pointer)
			if err != nil {
				return err
			}
			c = next
		}
		if previous, ok := a.namedCollectionUses[source.Canonical]; !ok || source.Pointer < previous {
			a.namedCollectionUses[source.Canonical] = source.Pointer
		}
	}
	return nil
}

// Capture node addresses in source order as well as serialized bytes. Equal
// replacement values must not acquire the original dictionary's authority.
func captureDictionaryAllOfNodes(root *document.Value, budget *dictionaryAllOfBudget, pointer string) ([]*document.Value, *diagnostic.Diagnostic) {
	var nodes []*document.Value
	stack := []*document.Value{root}
	for len(stack) != 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := budget.charge(1, 0, pointer); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		if node == nil {
			continue
		}
		switch node.Kind() {
		case document.Object:
			members := node.Members()
			if len(members) > 16384-budget.Work-len(stack) {
				return nil, diagnostic.At("E_MODEL_LIMIT", "Dictionary wrapper source capture exceeds the cumulative work or byte budget.", pointer)
			}
			for i := len(members) - 1; i >= 0; i-- {
				stack = append(stack, members[i].Value)
			}
		case document.Array:
			items := node.Items()
			if len(items) > 16384-budget.Work-len(stack) {
				return nil, diagnostic.At("E_MODEL_LIMIT", "Dictionary wrapper source capture exceeds the cumulative work or byte budget.", pointer)
			}
			for i := len(items) - 1; i >= 0; i-- {
				stack = append(stack, items[i])
			}
		}
	}
	return nodes, nil
}

// Only traverse already-bounded sources. Compare addresses directly while
// walking so replacement or additional nodes fail without allocating a copy.
func dictionaryAllOfNodesMatch(root *document.Value, expected []*document.Value) bool {
	stack := []*document.Value{root}
	index := 0
	for len(stack) != 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if index >= len(expected) || node != expected[index] {
			return false
		}
		index++
		if node == nil {
			continue
		}
		switch node.Kind() {
		case document.Object:
			members := node.Members()
			if len(members) > len(expected)-index-len(stack) {
				return false
			}
			for i := len(members) - 1; i >= 0; i-- {
				stack = append(stack, members[i].Value)
			}
		case document.Array:
			items := node.Items()
			if len(items) > len(expected)-index-len(stack) {
				return false
			}
			for i := len(items) - 1; i >= 0; i-- {
				stack = append(stack, items[i])
			}
		}
	}
	return index == len(expected)
}

func (a *analyzer) captureDictionaryAllOfDeclarations(declarations []Declaration) *diagnostic.Diagnostic {
	if len(a.dictionaryAllOfSources) == 0 {
		return nil
	}
	byName := map[string][]Declaration{}
	for _, declaration := range declarations {
		byName[declaration.Name()] = append(byName[declaration.Name()], declaration)
	}
	for _, source := range a.orderedDictionaryAllOfSources() {
		for i := range source.Bindings {
			binding := &source.Bindings[i]
			matches := byName[binding.Name]
			if len(matches) != 1 {
				return schemaReferenceUnsupported(source.Pointer)
			}
			alias, ok := matches[0].(*ValueAlias)
			if !ok {
				return schemaReferenceUnsupported(source.Pointer)
			}
			binding.Declaration = alias
			binding.Factory = alias.ComponentDefault
		}
	}
	return nil
}

func (a *analyzer) finalizeDictionaryAllOfDeclarations() {
	cache := map[*ValueAlias][]namedCollectionSourceType{}
	for _, source := range a.orderedDictionaryAllOfSources() {
		for i := range source.Bindings {
			binding := &source.Bindings[i]
			if types, ok := cache[binding.Declaration]; ok {
				binding.Types = types
				continue
			}
			seen := map[*TypeRef]bool{}
			for t := &binding.Declaration.Type; t != nil && !seen[t]; t = t.Element {
				seen[t] = true
				value := *t
				value.SchemaSteps = 0
				binding.Types = append(binding.Types, namedCollectionSourceType{t, value, namedCollectionChecks(value)})
			}
			cache[binding.Declaration] = binding.Types
		}
	}
}

func (a *analyzer) validateDictionaryAllOfSources(declarations []Declaration) *diagnostic.Diagnostic {
	if len(a.dictionaryAllOfSources) == 0 {
		return nil
	}
	byName := map[string][]Declaration{}
	rawByComponent := map[*component]string{}
	nodesByComponent := map[*component]bool{}
	physicalSchemas := map[string]*document.Value{}
	for _, entry := range a.componentSchemas.Members() {
		physicalSchemas[entry.Name] = entry.Value
	}
	for _, declaration := range declarations {
		byName[declaration.Name()] = append(byName[declaration.Name()], declaration)
	}
	for _, source := range a.orderedDictionaryAllOfSources() {
		root := source.Owner
		fail := func() *diagnostic.Diagnostic { return schemaReferenceUnsupported(source.Pointer) }
		if root.DictionaryAllOf != source || root.Schema != source.Schema ||
			!dictionaryAllOfMembersMatch(source) ||
			root.Reference != source.Text || root.BindingTarget != source.Canonical {
			return fail()
		}
		for _, binding := range source.Bindings {
			c := binding.Owner
			if c == nil || a.components[binding.Wire] != c || get(a.componentSchemas, binding.Wire) != binding.Schema ||
				physicalSchemas[binding.Wire] != binding.Schema || !slices.Equal(c.Schema.Members(), binding.Fields) ||
				c.Schema != binding.Schema || c.Wire != binding.Wire || c.Name != binding.Name ||
				c.Pointer != binding.Pointer || c.Reference != binding.Text || c.BindingTarget != binding.Target ||
				c.AliasHeight != binding.Height || c.BindingNullable != binding.Nullable ||
				c.BindingNullable != nil && *c.BindingNullable != binding.NullableValue {
				return fail()
			}
			raw, ok := rawByComponent[c]
			if !ok {
				data, _ := c.Schema.MarshalJSON()
				raw = string(data)
				rawByComponent[c] = raw
			}
			if !nodesByComponent[c] {
				if !dictionaryAllOfNodesMatch(c.Schema, binding.Nodes) {
					return fail()
				}
				nodesByComponent[c] = true
			}
			if raw != binding.Bytes {
				return fail()
			}
			ref := get(c.Schema, "$ref")
			if c.DictionaryAllOf != nil {
				wrapper := c.DictionaryAllOf
				if wrapper.Owner != c || wrapper.Schema != c.Schema || !dictionaryAllOfMembersMatch(wrapper) {
					return fail()
				}
				ref = get(wrapper.Member, "$ref")
			}
			if ref != binding.Reference {
				return fail()
			}
			matches := byName[binding.Name]
			if len(matches) != 1 || matches[0] != binding.Declaration ||
				binding.Declaration.ComponentDefault != binding.Factory {
				return fail()
			}
			for _, snapshot := range binding.Types {
				value := *snapshot.Node
				value.SchemaSteps = 0
				if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
					return fail()
				}
			}
		}
		if !dictionaryAllOfSameType(a.componentType(root), source.Bindings[0].Declaration.Type) {
			return fail()
		}
	}
	return nil
}

// Nullable() creates a fresh carrier each time. Compare that transient wrapper
// structurally while retaining the actual named backing and check addresses.
func dictionaryAllOfSameType(expected, actual TypeRef) bool {
	for expected.Kind == Nullable && actual.Kind == Nullable {
		if expected.Element == nil || actual.Element == nil {
			return false
		}
		left, right := expected, actual
		left.Element, right.Element = nil, nil
		left.SchemaSteps, right.SchemaSteps = 0, 0
		if left != right {
			return false
		}
		expected, actual = *expected.Element, *actual.Element
	}
	expected.SchemaSteps, actual.SchemaSteps = 0, 0
	return expected == actual
}
