package schema

import "openapi-kt/internal/document"

// Potential byte formats retain normal scalar grammar diagnostics before final
// profile eligibility. This is not payload validation or caller-type admission.
func nullableBytesCandidate(v *document.Value, kind string) bool {
	if kind != "string" || has(v, "enum") || has(v, "const") || has(v, "x-native-type") || has(v, "x-kotlin-value-class") || !has(v, "format") {
		return false
	}
	format := get(v, "format")
	return text(format) == "byte" || text(format) == "binary" || format.Kind() != document.String
}

// Byte/binary alias intent preserves the existing builtin array codec and
// Ktor's raw root projection. Caller classifiers are not source format profiles.
func bytesBinding(c *component) bool {
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || kind != "string" || nullable && !c.NullableBytes || c.ValueClass || c.Validation != nil || has(c.Schema, "enum") || has(c.Schema, "const") || has(c.Schema, "x-native-type") {
		return false
	}
	format := text(get(c.Schema, "format"))
	return c.Scalar == "kotlin.ByteArray" && (format == "byte" || format == "binary")
}
