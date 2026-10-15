package schema

import "openapi-kt/internal/document"

// Plain checked native enum roots own transparent nullable scalar intent. They
// remain separate from ordinary builtin, const, native and wrapper profiles.
func nullableNativeEnumCandidate(v *document.Value, kind string) bool {
	if !has(v, "enum") || kind != "integer" && kind != "number" && kind != "boolean" {
		return false
	}
	for _, key := range []string{"const", "x-native-type", "x-kotlin-value-class", "allOf", "oneOf", "anyOf", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	return true
}
