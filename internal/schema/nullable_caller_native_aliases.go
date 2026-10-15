package schema

import "openapi-kt/internal/document"

func nullableCallerNativeCandidate(v *document.Value, kind string) bool {
	if !has(v, "x-native-type") || has(v, "enum") || has(v, "const") || has(v, "x-kotlin-value-class") {
		return false
	}
	native := text(get(v, "x-native-type"))
	if native == "date" || native == "bigint" {
		return false
	}
	return kind == "native" || kind == "string" || kind == "integer" || kind == "number" || kind == "boolean"
}

// Caller-native bindings retain the original source profile, not a coincident
// builtin classifier name. Declaration/serializer ownership stays with callers.
func callerNativeBinding(c *component) bool {
	_, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable && !c.NullableCallerNative || c.ValueClass || c.Validation != nil || has(c.Schema, "enum") || has(c.Schema, "const") || !has(c.Schema, "x-native-type") {
		return false
	}
	native := text(get(c.Schema, "x-native-type"))
	return native != "" && native != "date" && native != "bigint" && c.Scalar == native
}
