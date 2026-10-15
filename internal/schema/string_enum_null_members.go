package schema

import "openapi-kt/internal/document"

// Open String enums retain their non-null physical vocabulary classifier. An
// admitted null member uses the existing explicit nullable typed view only.
func nullableStringEnumNulls(v *document.Value, pointer string) bool {
	kind, nullable, err := schemaType(v, pointer)
	return err == nil && nullable && (kind == "" || kind == "string") && plainStringEnumSchema(v)
}
