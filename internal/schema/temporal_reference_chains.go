package schema

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Temporal aliases retain the actual transparent scalar declaration and source.
// Unlike a nominal DTO or wrapper, this mapping never ends source-hop accounting.
type temporalOrigin struct {
	Owner           *component
	Schema          *document.Value
	Pointer, Scalar string
}

func (a *analyzer) recordTemporalOrigin(d Declaration, c *component) {
	if a.temporalOrigins != nil && temporalBinding(c) {
		a.temporalOrigins[d] = temporalOrigin{c, c.Schema, c.Pointer, c.Scalar}
	}
}

func (a *analyzer) bindNamedChainTemporal(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalTemporal
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		}
	}
	if canonical.Schema != target.Schema || !temporalBinding(canonical) {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.temporalOrigins {
			if origin.Owner == canonical && origin.Schema == target.Schema && origin.Pointer == canonical.Pointer && origin.Scalar == canonical.Scalar && temporalDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalTemporal = canonical
	if a.temporalUses == nil {
		a.temporalUses = map[*component]string{}
	}
	if previous, ok := a.temporalUses[canonical]; !ok || source < previous {
		a.temporalUses[canonical] = source
	}
	return true
}

func temporalDeclaration(d Declaration, origin temporalOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullableTemporal && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullableTemporal && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar
	}
	return false
}

func (a *analyzer) validateTemporalBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]bool{}
	for _, d := range declarations {
		actual[d] = true
	}
	used := []*component{}
	for c := range a.temporalUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.temporalUses[used[i]] < a.temporalUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.temporalOrigins {
			if actual[d] && origin.Owner == c && origin.Schema == c.Schema && origin.Pointer == c.Pointer && origin.Scalar == c.Scalar && temporalBinding(c) && temporalDeclaration(d, origin) {
				matches++
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.temporalUses[c])
		}
	}
	return nil
}
