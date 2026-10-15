package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
)

// Bytes is a source marker, not ByteArray spelling. Its canonical scalar remains
// the provider ByteArray mapping; JSON signed arrays and raw HTTP stay separate.
type bytesOrigin struct {
	Owner                        *component
	Schema                       *document.Value
	Pointer, Scalar, SourceBytes string
}

func (a *analyzer) recordBytesOrigin(d Declaration, c *component) {
	if !bytesBinding(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.bytesOrigins[d] = bytesOrigin{c, c.Schema, c.Pointer, c.Scalar, string(bytes)}
}

func (a *analyzer) bindNamedChainBytes(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalBytes
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		}
	}
	if canonical.Schema != target.Schema || canonical.Pointer != target.Pointer || !bytesBinding(canonical) {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.bytesOrigins {
			if origin.Owner == canonical && bytesSourceMatches(origin) && bytesDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalBytes = canonical
	if previous, ok := a.bytesUses[canonical]; !ok || source < previous {
		a.bytesUses[canonical] = source
	}
	return true
}

func bytesSourceMatches(origin bytesOrigin) bool {
	c := origin.Owner
	if origin.Schema != c.Schema || origin.Pointer != c.Pointer || origin.Scalar != c.Scalar || !bytesBinding(c) {
		return false
	}
	bytes, _ := c.Schema.MarshalJSON()
	return string(bytes) == origin.SourceBytes
}

func bytesDeclaration(d Declaration, origin bytesOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullableBytes && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullableBytes && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar && bare(d.Type).Validation == nil && bare(d.Type).Serializer == ""
	}
	return false
}

func (a *analyzer) validateBytesBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.bytesUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.bytesUses[used[i]] < a.bytesUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.bytesOrigins {
			if actual[d] != 0 && origin.Owner == c && bytesSourceMatches(origin) && bytesDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.bytesUses[c])
		}
	}
	return nil
}
