package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type dtoUniqueContainerInitializer struct {
	Literal     string
	DefaultNull bool
}

// This first collection-field role follows only original inline schema edges.
// Field defaults are local; references and nominal/composed leaves are separate.
func (a *analyzer) dtoUniqueContainerSource(v *document.Value, p string, depth int, root bool) bool {
	ok, err := a.dtoUniqueContainerCandidate(v, p, depth, root)
	return ok && err == nil
}

// Preserve original null-leaf diagnostics during DTO-key preflight. A malformed
// vocabulary/default must not be hidden behind the consuming uniqueItems role.
func (a *analyzer) dtoUniqueContainerCandidate(v *document.Value, p string, depth int, root bool) (bool, *diagnostic.Diagnostic) {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false, nil
	}
	for _, key := range []string{"$ref", "oneOf", "anyOf", "allOf", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false, nil
		}
	}
	kind, _, e := schemaType(v, p)
	if e != nil {
		return false, nil
	}
	edge := ""
	if kind == "array" {
		edge = "items"
	} else if dictionarySchema(v) && (kind == "" || kind == "object") {
		edge = "additionalProperties"
	}
	if edge != "" {
		keys := a.collectionKeys([]string{"type", "nullable", edge})
		if root {
			keys = append(keys, "default")
		}
		if checkKeys(v, keys, p) != nil {
			return false, nil
		}
		return a.dtoUniqueContainerCandidate(get(v, edge), naming.Pointer(p, edge), depth+1, false)
	}
	if has(v, "enum") && kind != "null" {
		if !root && lexicalStringEnumSchema(v) && (kind == "string" || kind == "") {
			_, err := buildStringEnumKeys("DTOUnique", v, p, false)
			return err == nil, err
		}
		if !root && a.options.Validation && (kind == "integer" || kind == "number" || kind == "boolean") {
			probe := *a
			probe.warnings = nil
			probe.validationSources = map[*PrimitiveValidation]string{}
			probe.emissionHints = EmissionHints{}
			_, _, err := probe.nativeEnumFieldType(field{Schema: v, Pointer: p, Depth: depth}, false)
			return err == nil, err
		}
		return false, nil
	}
	if !root && kind == "null" {
		if has(v, "enum") {
			_, err := nullOnlyEnumProfile(v, p, false)
			return err == nil, err
		}
		err := checkKeys(v, []string{"type", "nullable"}, p)
		return err == nil, err
	}
	return !root && a.uniqueScalarSource(v, p, depth), nil
}

func dtoUniqueContainerType(t TypeRef) bool {
	t = defaultBacking(t).WithoutRootNull()
	return t.Kind == List || t.Kind == Map
}

func dtoUniqueFieldDefaultKey(t TypeRef, v *document.Value) string {
	if dtoUniqueContainerType(t) {
		return structuralUniqueDefaultKey(t, v)
	}
	return uniqueScalarDefaultKey(t, v)
}

// Count container nodes, scalar/null values and map keys before normalization.
// The original default walker has already checked value depth and size.
func dtoUniqueLiteralWork(v *document.Value) int {
	if v == nil {
		return 1
	}
	work := 1
	if v.Kind() == document.Array {
		for _, child := range v.Items() {
			work += dtoUniqueLiteralWork(child)
		}
	}
	if v.Kind() == document.Object {
		for _, member := range v.Members() {
			work += 1 + dtoUniqueLiteralWork(member.Value)
		}
	}
	return work
}

// This local DTO-getter role admits JsonNull leaves without granting structural
// array uniqueness or reference/projection permissions to other planners.
func dtoUniqueContainerTreeType(t TypeRef, depth int) bool {
	if depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	if t.Kind == Nullable {
		return t.Element != nil && dtoUniqueContainerTreeType(*t.Element, depth)
	}
	if t.Kind == List || t.Kind == Map {
		return t.Element != nil && dtoUniqueContainerTreeType(*t.Element, depth+1)
	}
	return dtoUniqueFieldScalarType(t) || uniqueStringEnumType(t)
}
func (a *analyzer) dtoUniqueContainerTypeMatches(t TypeRef, v *document.Value, p string, depth int) bool {
	if v == nil || depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	for t.Kind == Nullable {
		if t.Element == nil {
			return false
		}
		t = *t.Element
	}
	kind, _, err := schemaType(v, p)
	if err != nil {
		return false
	}
	if kind == "array" {
		return t.Kind == List && t.Element != nil && a.dtoUniqueContainerTypeMatches(*t.Element, get(v, "items"), naming.Pointer(p, "items"), depth+1)
	}
	if dictionarySchema(v) {
		return t.Kind == Map && t.Element != nil && a.dtoUniqueContainerTypeMatches(*t.Element, get(v, "additionalProperties"), naming.Pointer(p, "additionalProperties"), depth+1)
	}
	if kind == "null" {
		return t.Kind == Scalar && t.Name == "kotlinx.serialization.json.JsonNull" && t.Element == nil && a.dtoUniqueContainerSource(v, p, depth, false)
	}
	return a.structuralUniqueTypeMatches(t, v, p, depth)
}

// Classify only an already-admitted original inline tree. A String vocabulary
// contributes an abstract serialName key, never a provisional declaration.
func dtoUniqueStringEnumTreeSource(v *document.Value) bool {
	if v == nil {
		return false
	}
	if has(v, "items") {
		return dtoUniqueStringEnumTreeSource(get(v, "items"))
	}
	if dictionarySchema(v) {
		return dtoUniqueStringEnumTreeSource(get(v, "additionalProperties"))
	}
	kind, _, err := schemaType(v, "")
	return err == nil && has(v, "enum") && (kind == "string" || kind == "")
}
func dtoUniqueStringEnumTreeType(v *document.Value, p string, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, p); err != nil {
		return TypeRef{}, err
	}
	kind, nullable, err := schemaType(v, p)
	if err != nil {
		return TypeRef{}, err
	}
	var ty TypeRef
	if kind == "array" || dictionarySchema(v) {
		edge, k := "items", List
		if dictionarySchema(v) {
			edge, k = "additionalProperties", Map
		}
		child, e := dtoUniqueStringEnumTreeType(get(v, edge), naming.Pointer(p, edge), depth+1)
		if e != nil {
			return TypeRef{}, e
		}
		ty = TypeRef{Kind: k, Element: &child}
	} else {
		if _, e := buildStringEnumKeys("DTOUnique", v, p, false); e != nil {
			return TypeRef{}, e
		}
		ty = TypeRef{Kind: Named, Name: dtoUniqueEnumKind}
	}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

func (a *analyzer) dtoUniqueContainerEnumsMatch(model *Model, parent string, t TypeRef, v *document.Value, p string, src *dtoUniqueSource, depth int) bool {
	if v == nil || depth > MaxSchemaDepth {
		return false
	}
	for t.Kind == Nullable {
		if t.Element == nil {
			return false
		}
		t = *t.Element
	}
	kind, _, err := schemaType(v, p)
	if err != nil {
		return false
	}
	if kind == "array" || dictionarySchema(v) {
		edge := "items"
		if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		return t.Element != nil && a.dtoUniqueContainerEnumsMatch(model, parent, *t.Element, get(v, edge), naming.Pointer(p, edge), src, depth+1)
	}
	if has(v, "enum") && (kind == "string" || kind == "") {
		return a.dtoUniqueEnumField(model, parent, t, field{Schema: v, Pointer: p}, src)
	}
	return true
}
