package schema

import (
	"encoding/json"
	"fmt"
	"sort"
)

// DeduplicateOperationModels folds the initial ordinary DTO/enum subset after
// complete analysis. Work/diagnostics were checked on every original occurrence.
// Components and external nominal types remain opaque, not structural peers.
func DeduplicateOperationModels(plan *Plan, packageName string) {
	if plan.Client == nil {
		return
	}
	protected := operationOpaqueOwners(plan, packageName)
	declarations := append([]Declaration(nil), plan.Client.Models...)
	sort.Slice(declarations, func(i, j int) bool { return declarations[i].Name() < declarations[j].Name() })
	canonical := map[string]Declaration{}
	replacements := map[Declaration]Declaration{}
	for _, d := range declarations {
		if protected[d.Name()] || !dedupEligible(d) {
			continue
		}
		key := operationEquivalenceKey(d, packageName)
		if previous := canonical[key]; previous != nil {
			plan.EmissionHints.transferEquivalent(previous, d)
			replacements[d] = plan.ownedAlias(d.Name(), packageName+".api.models."+previous.Name())
		} else {
			canonical[key] = d
		}
	}
	for i, d := range plan.Client.Models {
		if alias := replacements[d]; alias != nil {
			plan.Client.Models[i] = alias
		}
	}
}

func operationEquivalenceKey(d Declaration, packageName string) string {
	names := map[string]string{}
	dedupOwnedNames(d, d.Name(), "$root", packageName+".api.models", names)
	encoded, err := json.Marshal(dedupView(d, names, true))
	if err != nil {
		panic("invalid internal operation equivalence view")
	}
	return fmt.Sprintf("%T:%s", d, encoded)
}

func dedupEligible(d Declaration) bool {
	switch d := d.(type) {
	case *StringEnum:
		return true
	case *Model:
		// Empty/raw projections already have no consuming DTO identity to fold.
		if len(d.Properties) == 0 && len(d.Constants) == 0 {
			return false
		}
		for _, n := range d.Nested {
			if !dedupEligible(n) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Only declared owned classifier spellings normalize. Native Scalar names and
// serializer/constant expressions remain opaque even if they resemble owners.
func dedupOwnedNames(d Declaration, path, normalized, pkg string, names map[string]string) {
	names[path] = normalized
	names[pkg+"."+path] = normalized
	if m, ok := d.(*Model); ok {
		for _, n := range m.Nested {
			dedupOwnedNames(n, path+"."+n.Name(), normalized+"."+n.Name(), pkg, names)
		}
	}
}
func dedupType(t TypeRef, names map[string]string) TypeRef {
	t.SchemaSteps = 0 // Work was checked before this presentation-only comparison.
	if t.Kind == Named {
		if name, ok := names[t.Name]; ok {
			t.Name = name
		}
	}
	if t.Element != nil {
		element := dedupType(*t.Element, names)
		t.Element = &element
	}
	return t
}
func dedupView(d Declaration, names map[string]string, top bool) Declaration {
	switch d := d.(type) {
	case *StringEnum:
		view := *d
		if top {
			view.ClassName = "$root"
		}
		return &view
	case *Model:
		view := *d
		if top {
			view.ClassName = "$root"
		}
		view.Properties = append([]Property(nil), d.Properties...)
		for i := range view.Properties {
			view.Properties[i].Type = dedupType(view.Properties[i].Type, names)
		}
		view.Constants = append([]Constant(nil), d.Constants...)
		for i := range view.Constants {
			view.Constants[i].Type = dedupType(view.Constants[i].Type, names)
		}
		view.Nested = make([]Declaration, len(d.Nested))
		for i, n := range d.Nested {
			view.Nested[i] = dedupView(n, names, false)
		}
		return &view
	default:
		panic("ineligible operation equivalence view")
	}
}
