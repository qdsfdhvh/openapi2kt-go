package schema

import "openapi-kt/internal/document"

// Inline values have a lexical owner in each real variant. A common property
// cannot refer to both owners, even when their fields happen to be identical.
// Classify only original inline syntax here; normal variant planning validates
// all source fields, owns their declarations and enforces source/work limits.
func inlineVariantValue(f *field) bool {
	if has(f.Schema, "$ref") {
		return false // Original named references use actual shared declarations.
	}
	if f.Depth > MaxSchemaDepth {
		return true // Real branch planning reports the original depth diagnostic.
	}
	v := f.Schema
	if has(v, "default") && has(v, "enum") {
		kind, _, _ := schemaType(v, f.Pointer)
		if kind == "" || kind == "string" {
			return true
		}
	}
	if has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") {
		return true
	}
	if properties := get(v, "properties"); properties != nil && properties.Kind() == document.Object && len(properties.Members()) > 0 {
		return true
	}
	for _, key := range []string{"items", "additionalProperties"} {
		if child, ok := v.Get(key); ok && inlineVariantValue(&field{Schema: child, Depth: f.Depth + 1}) {
			return true
		}
	}
	return false
}
