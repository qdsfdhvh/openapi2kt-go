package schema

import "openapi-kt/internal/document"

// Intrinsic nullability changes typed uses, not the canonical vocabulary class.
// Grammar and unique string members are still validated by buildStringEnum.
func plainStringEnumSchema(v *document.Value) bool {
	if !has(v, "enum") {
		return false
	}
	for _, key := range []string{"format", "x-native-type", "x-kotlin-value-class", "const", "allOf", "oneOf", "anyOf", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	return true
}

func plainStringEnumBinding(c *component) bool {
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	return err == nil && (!nullable || c.NullableEnum) && (kind == "" || kind == "string") && plainStringEnumSchema(c.Schema) && !c.ValueClass && c.Scalar == ""
}
