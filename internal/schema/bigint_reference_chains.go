package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
)

// Bigint is a source marker, not Long spelling. Its canonical scalar remains
// unannotated; only real field uses attach the separate string serializer.
type bigintOrigin struct {
	Owner                        *component
	Schema                       *document.Value
	Pointer, Scalar, SourceBytes string
}

func (a *analyzer) recordBigintOrigin(d Declaration, c *component) {
	if !bigintBinding(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.bigintOrigins[d] = bigintOrigin{c, c.Schema, c.Pointer, c.Scalar, string(bytes)}
}

func (a *analyzer) bindNamedChainBigint(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalBigint
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		}
	}
	if canonical.Schema != target.Schema || canonical.Pointer != target.Pointer || !bigintBinding(canonical) {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.bigintOrigins {
			if origin.Owner == canonical && bigintSourceMatches(origin) && bigintDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalBigint = canonical
	if previous, ok := a.bigintUses[canonical]; !ok || source < previous {
		a.bigintUses[canonical] = source
	}
	return true
}

func bigintSourceMatches(origin bigintOrigin) bool {
	c := origin.Owner
	if origin.Schema != c.Schema || origin.Pointer != c.Pointer || origin.Scalar != c.Scalar || !bigintBinding(c) {
		return false
	}
	bytes, _ := c.Schema.MarshalJSON()
	return string(bytes) == origin.SourceBytes
}

func bigintDeclaration(d Declaration, origin bigintOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullableBigint && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullableBigint && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar && bare(d.Type).Validation == nil && bare(d.Type).Serializer == ""
	}
	return false
}

func (a *analyzer) validateBigintBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.bigintUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.bigintUses[used[i]] < a.bigintUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.bigintOrigins {
			if actual[d] != 0 && origin.Owner == c && bigintSourceMatches(origin) && bigintDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.bigintUses[c])
		}
	}
	return nil
}
