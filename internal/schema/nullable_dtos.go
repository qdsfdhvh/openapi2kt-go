package schema

import "openapi-kt/internal/document"

// Raw eligibility is syntactic only; original grammar validation and the final
// empty ordinary Model check still run before exposing a transparent alias.
func nullableRawCandidate(v *document.Value) bool {
	if dictionarySchema(v) {
		return false
	}
	for _, key := range []string{"allOf", "oneOf", "anyOf", "discriminator", "enum", "const", "x-native-type", "x-kotlin-value-class", "additionalProperties"} {
		if has(v, key) {
			return false
		}
	}
	properties := get(v, "properties")
	return !has(v, "properties") || properties.Kind() == document.Object && len(properties.Members()) == 0
}

// Nullable roots expose the existing ordinary classifier, not a second model.
// Final planning still requires a constructor property; syntactic properties
// may contain only constants or projections, which do not establish eligibility.
func nullableDTOCandidate(v *document.Value) bool {
	properties := get(v, "properties")
	if dictionarySchema(v) || !has(v, "allOf") && (properties.Kind() != document.Object || len(properties.Members()) == 0) {
		return false
	}
	// allOf syntax grants only provisional nullable DTO classification. Normal
	// shape planning and the final original Model constructor check authorize
	// the physical type; empty/const-only/product roots retain their rejection.
	for _, key := range []string{"oneOf", "anyOf", "discriminator", "enum", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	return true
}
