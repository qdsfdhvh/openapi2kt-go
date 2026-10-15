package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Named native enum sources are transparent scalars. Their original catalog,
// vocabulary and actual export authorize typed checks through borrowed graphs.
type nativeEnumOrigin struct {
	Owner                          *component
	Schema, Vocabulary             *document.Value
	Members                        []*document.Value
	SchemaMembers                  []document.Member
	Pointer, Bytes, Scalar, Checks string
	Validation                     *PrimitiveValidation
	Actual                         Declaration
}

func nativeEnumBinding(c *component) bool {
	if c == nil || c.Reference != "" || c.ValueClass || has(c.Schema, "const") || has(c.Schema, "x-native-type") || has(c.Schema, "x-kotlin-value-class") || !has(c.Schema, "enum") || c.Validation == nil || len(c.Validation.Enum) == 0 {
		return false
	}
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	return err == nil && (!nullable || c.NullableNativeEnum) && (kind == "integer" || kind == "number" || kind == "boolean")
}

func (a *analyzer) recordNativeEnumOrigin(d Declaration, c *component) {
	if !a.options.Validation || !nativeEnumBinding(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.nativeEnumOrigins[c] = &nativeEnumOrigin{Owner: c, Schema: c.Schema, Pointer: c.Pointer, Bytes: string(bytes), Scalar: c.Scalar, Validation: c.Validation, Checks: namedCollectionChecks(TypeRef{Validation: c.Validation}), Vocabulary: get(c.Schema, "enum"), Members: append([]*document.Value{}, get(c.Schema, "enum").Items()...), SchemaMembers: append([]document.Member{}, c.Schema.Members()...), Actual: d}
}

func (a *analyzer) bindNamedChainNativeEnum(target *resolvedSchema, source string) bool {
	if !a.options.Validation || target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalNativeEnum
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		}
	}
	if !nativeEnumBinding(canonical) || target.Schema != canonical.Schema || target.Pointer != canonical.Pointer {
		return false
	}
	if a.nominalsLinked {
		origin := a.nativeEnumOrigins[canonical]
		if origin == nil || origin.Schema != canonical.Schema || origin.Validation != canonical.Validation || !nativeEnumDeclaration(origin.Actual, origin) {
			return false
		}
	}
	if target.Nullable == nil {
		target.Nullable = nullableOverride(target.Catalog.Schema)
		if target.Nullable == nil && canonical.NullableNativeEnum {
			flag := true
			target.Nullable = &flag
		}
	}
	target.CanonicalNativeEnum = canonical
	target.Steps = max(target.Steps, 1+target.Catalog.AliasHeight)
	if previous, ok := a.nativeEnumUses[canonical]; !ok || source < previous {
		a.nativeEnumUses[canonical] = source
	}
	return true
}

func nativeEnumDeclaration(d Declaration, origin *nativeEnumOrigin) bool {
	if d == nil || d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullableNativeEnum && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullableNativeEnum && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar && bare(d.Type).Validation == origin.Validation
	}
	return false
}

func (a *analyzer) validateNativeEnumBindings(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.nativeEnumUses) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	used := []*component{}
	for c := range a.nativeEnumUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.nativeEnumUses[used[i]] < a.nativeEnumUses[used[j]] })
	for _, c := range used {
		source := a.nativeEnumUses[c]
		origin := a.nativeEnumOrigins[c]
		if origin == nil || a.components[c.Wire] != c || origin.Owner != c || origin.Schema != c.Schema || origin.Pointer != c.Pointer || origin.Scalar != c.Scalar || origin.Validation != c.Validation || !nativeEnumBinding(c) || actual[origin.Actual] != 1 || !nativeEnumDeclaration(origin.Actual, origin) || get(c.Schema, "enum") != origin.Vocabulary || !slices.Equal(origin.Members, origin.Vocabulary.Items()) || !slices.Equal(origin.SchemaMembers, c.Schema.Members()) || namedCollectionChecks(TypeRef{Validation: c.Validation}) != origin.Checks {
			return schemaReferenceUnsupported(source)
		}
		for _, member := range origin.SchemaMembers {
			if get(c.Schema, member.Name) != member.Value {
				return schemaReferenceUnsupported(source)
			}
		}
		bytes, _ := c.Schema.MarshalJSON()
		if string(bytes) != origin.Bytes {
			return schemaReferenceUnsupported(source)
		}
		matches := 0
		for _, d := range ds {
			if d.Name() == c.Name {
				matches++
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(source)
		}
	}
	return nil
}
