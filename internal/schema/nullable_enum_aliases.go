package schema

import "openapi-kt/internal/document"

// Intrinsic nullability changes typed uses, not the canonical vocabulary class.
// Grammar and unique string members are still validated by buildStringEnum.
func plainStringEnumSchema(v *document.Value) bool {
	return lexicalStringEnumSchema(v) && !has(v, "format") && !has(v, "x-native-type")
}

// Metadata is validated by the original lexical vocabulary builder. It does
// not turn a String enum into a date, byte array or caller-owned scalar codec.
// Null-member inference deliberately continues using the plain profile above.
func lexicalStringEnumSchema(v *document.Value) bool {
	if !has(v, "enum") {
		return false
	}
	for _, key := range []string{"x-kotlin-value-class", "const", "allOf", "oneOf", "anyOf", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	return true
}

func lexicalStringEnumBinding(c *component) bool {
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	return err == nil && (!nullable || c.NullableEnum) && (kind == "" || kind == "string") && lexicalStringEnumSchema(c.Schema) && !c.ValueClass && c.Scalar == ""
}

// Metadata does not replace a model enum's lexical vocabulary codec. The
// shared enum builder validates keys,
// formats, native spellings and every original vocabulary member.
func metadataStringEnumCandidate(v *document.Value) bool {
	if !has(v, "enum") || has(v, "$ref") || !has(v, "format") && !has(v, "x-native-type") {
		return false
	}
	kind, _, err := schemaType(v, "")
	return err == nil && (kind == "" || kind == "string")
}
