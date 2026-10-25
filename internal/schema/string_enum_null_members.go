package schema

import "openapi-kt/internal/document"

// Open String enums retain their non-null physical vocabulary classifier. An
// admitted null member uses the existing nullable typed view only.
func nullableStringEnumNulls(v *document.Value, pointer string) bool {
	kind, nullable, err := schemaType(v, pointer)
	return err == nil && nullable && (kind == "" || kind == "string") && plainStringEnumSchema(v)
}

// Only a typeless plain String vocabulary supplies implicit null intent.
// Full grammar, uniqueness and raw member limits stay with the enum planner.
func implicitStringEnumNull(v *document.Value) bool {
	if !plainStringEnumSchema(v) {
		return false
	}
	for _, key := range []string{"type", "nullable", "$ref", "properties", "additionalProperties", "required", "items", "prefixItems"} {
		if has(v, key) {
			return false
		}
	}
	values := get(v, "enum")
	if values.Kind() != document.Array || len(values.Items()) > MaxMembers {
		return false
	}
	hasString, hasNull := false, false
	for _, value := range values.Items() {
		switch value.Kind() {
		case document.String:
			hasString = true
		case document.Null:
			hasNull = true
		default:
			return false
		}
	}
	return hasString && hasNull
}
