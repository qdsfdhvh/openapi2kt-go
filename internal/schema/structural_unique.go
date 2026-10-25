package schema

import (
	"encoding/json"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Structural equality requires an actual finite typed List/Map tree and the
// original accepted leaf source. Neither a projected spelling nor a nominal
// DTO obtains equality just because it occupies a collection element.
func structuralUniqueType(t TypeRef, depth int) bool {
	for t.Kind == Nullable || t.Kind == Named && t.Element != nil {
		if depth > MaxSchemaDepth {
			return false
		}
		if t.Kind == Named {
			depth++
		}
		t = *t.Element
	}
	return (t.Kind == List || t.Kind == Map) && structuralUniqueTreeType(t, depth)
}

func structuralUniqueTreeType(t TypeRef, depth int) bool {
	if depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	if t.Kind == Nullable || t.Kind == Named && t.Element != nil {
		if t.Kind == Named {
			depth++
		}
		return structuralUniqueTreeType(*t.Element, depth)
	}
	if t.Kind == List || t.Kind == Map {
		return t.Element != nil && structuralUniqueTreeType(*t.Element, depth+1)
	}
	return uniqueScalarType(t) || uniqueStringEnumType(t)
}

func (a *analyzer) structuralUniqueSource(v *document.Value, pointer string, depth int, active map[*document.Value]bool, root bool) bool {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth || active[v] {
		return false
	}
	active[v] = true
	defer delete(active, v)
	if has(v, "$ref") {
		if checkKeys(v, []string{"$ref", "nullable"}, pointer) != nil {
			return false
		}
		target, err := a.resolveSchema(text(get(v, "$ref")), naming.Pointer(pointer, "$ref"))
		return err == nil && a.structuralUniqueSource(target.Schema, target.Pointer, depth+max(1, target.Steps), active, root)
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return false
	}
	edge := ""
	if kind == "array" {
		edge = "items"
	} else if dictionarySchema(v) && (kind == "" || kind == "object") {
		edge = "additionalProperties"
	}
	if edge != "" {
		keys := a.collectionKeys([]string{"type", "nullable", edge})
		if a.componentDefaultSchemas[v] {
			keys = append(keys, "default")
		}
		if checkKeys(v, keys, pointer) != nil {
			return false
		}
		return a.structuralUniqueSource(get(v, edge), naming.Pointer(pointer, edge), depth+1, active, false)
	}
	return !root && (a.uniqueScalarSource(v, pointer, depth) || a.uniqueNativeEnumSource(v, pointer, depth) || a.uniqueStringEnumSource(v, pointer, depth))
}

type structuralUniqueUse struct {
	Element       *TypeRef
	Pointer       string
	Schema        *document.Value
	SourcePointer string
}

// Forward collection aliases share their real backing TypeRef. Validate that
// backing after all declarations and operation consumers have been linked.
// The shared queue also retains discarded/scratch checks in original order.
func (a *analyzer) validateStructuralUniqueUses() *diagnostic.Diagnostic {
	if a.structuralUniqueUses != nil {
		for _, use := range *a.structuralUniqueUses {
			if !structuralUniqueType(*use.Element, 0) || !a.structuralUniqueTypeMatches(*use.Element, use.Schema, use.SourcePointer, 0) {
				return unsupported(use.Pointer)
			}
		}
	}
	return nil
}

func (a *analyzer) structuralUniqueTypeMatches(t TypeRef, v *document.Value, pointer string, depth int) bool {
	if depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	for t.Kind == Nullable || t.Kind == Named && t.Element != nil {
		t = *t.Element
	}
	if has(v, "$ref") {
		target, err := a.resolveSchema(text(get(v, "$ref")), naming.Pointer(pointer, "$ref"))
		return err == nil && a.structuralUniqueTypeMatches(t, target.Schema, target.Pointer, depth+max(1, target.Steps))
	}
	kind, _, _ := schemaType(v, pointer)
	if kind == "array" {
		return t.Kind == List && t.Element != nil && a.structuralUniqueTypeMatches(*t.Element, get(v, "items"), naming.Pointer(pointer, "items"), depth+1)
	}
	if dictionarySchema(v) {
		return t.Kind == Map && t.Element != nil && a.structuralUniqueTypeMatches(*t.Element, get(v, "additionalProperties"), naming.Pointer(pointer, "additionalProperties"), depth+1)
	}
	if union, err := a.readUnion(v, pointer); err == nil && union != nil && union.HasNull && len(union.Branches) == 1 {
		return a.structuralUniqueTypeMatches(t, union.Branches[0].Schema, union.Branches[0].Pointer, depth+1)
	}
	if has(v, "enum") {
		return a.uniqueStringEnumSource(v, pointer, depth) && uniqueStringEnumType(t) ||
			a.uniqueNativeEnumSource(v, pointer, depth) && uniqueNativeEnumType(t)
	}
	return uniqueScalarType(t) && a.uniqueScalarSource(v, pointer, depth)
}

func structuralUniqueCollectionDepth(t TypeRef) int {
	if t.Element == nil {
		return 0
	}
	depth := structuralUniqueCollectionDepth(*t.Element)
	if t.Kind == List || t.Kind == Map {
		depth++
	}
	return depth
}

// Called only after the ordinary annotation walker has validated every leaf.
// Keep keys separate from emitted JSON, preserving original lexemes/Pointers.
func structuralUniqueDefaultKey(t TypeRef, v *document.Value) string {
	encoded, _ := json.Marshal(structuralUniqueDefaultValue(t, v))
	return string(encoded)
}

// Encode the complete key once. Encoding child keys as strings at every level
// would repeatedly escape backslashes and grow exponentially with tree depth.
func structuralUniqueDefaultValue(t TypeRef, v *document.Value) any {
	if v.Kind() == document.Null {
		return nil
	}
	for t.Kind == Nullable || t.Kind == Named && t.Element != nil {
		t = *t.Element
	}
	switch t.Kind {
	case List:
		keys := make([]any, 0, len(v.Items()))
		for _, member := range v.Items() {
			keys = append(keys, structuralUniqueDefaultValue(*t.Element, member))
		}
		return keys
	case Map:
		keys := make(map[string]any, len(v.Members()))
		for _, member := range v.Members() {
			keys[member.Name] = structuralUniqueDefaultValue(*t.Element, member.Value)
		}
		return keys
	default:
		return uniqueScalarDefaultKey(t, v)
	}
}
