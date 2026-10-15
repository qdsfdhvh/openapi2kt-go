package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
)

// Caller-native authority belongs to the original native string/schema and
// generator alias. The provided class, serializer and layout remain caller-owned.
type callerNativeOrigin struct {
	Owner                        *component
	Schema                       *document.Value
	Pointer, Scalar, SourceBytes string
}

func (a *analyzer) recordCallerNativeOrigin(d Declaration, c *component) {
	if !callerNativeBinding(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.callerNativeOrigins[d] = callerNativeOrigin{c, c.Schema, c.Pointer, c.Scalar, string(bytes)}
}

func (a *analyzer) bindNamedChainCallerNative(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalCallerNative
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		}
	}
	if canonical.Schema != target.Schema || canonical.Pointer != target.Pointer || !callerNativeBinding(canonical) {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.callerNativeOrigins {
			if origin.Owner == canonical && callerNativeSourceMatches(origin) && callerNativeDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalCallerNative = canonical
	if previous, ok := a.callerNativeUses[canonical]; !ok || source < previous {
		a.callerNativeUses[canonical] = source
	}
	return true
}

func callerNativeSourceMatches(origin callerNativeOrigin) bool {
	c := origin.Owner
	if origin.Schema != c.Schema || origin.Pointer != c.Pointer || origin.Scalar != c.Scalar || !callerNativeBinding(c) {
		return false
	}
	bytes, _ := c.Schema.MarshalJSON()
	return string(bytes) == origin.SourceBytes
}

func callerNativeDeclaration(d Declaration, origin callerNativeOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullableCallerNative && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullableCallerNative && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar && bare(d.Type).Validation == nil && bare(d.Type).Serializer == ""
	}
	return false
}

func (a *analyzer) validateCallerNativeBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.callerNativeUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.callerNativeUses[used[i]] < a.callerNativeUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.callerNativeOrigins {
			if actual[d] != 0 && origin.Owner == c && callerNativeSourceMatches(origin) && callerNativeDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.callerNativeUses[c])
		}
	}
	return nil
}
