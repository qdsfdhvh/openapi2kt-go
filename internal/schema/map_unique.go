package schema

import "openapi-kt/internal/document"

// Only direct typed dictionaries with ordinary builtin value codecs qualify.
// Neither a projected Map spelling nor a reference grants structural equality.
func uniqueMapType(ty TypeRef) bool {
	ty = ty.WithoutRootNull()
	if ty.Kind != Map || ty.Serializer != "" || ty.Element == nil {
		return false
	}
	value := ty.Element.WithoutRootNull()
	return value.Kind == Scalar && value.Serializer == "" &&
		(value.Name == "kotlin.String" || value.Name == "kotlin.Int" || value.Name == "kotlin.Long" || value.Name == "kotlin.Boolean" ||
			value.Name == "kotlin.uuid.Uuid" || value.Name == "kotlinx.datetime.LocalDate" || value.Name == "kotlin.time.Instant")
}

func (a *analyzer) uniqueMapSource(v *document.Value, pointer string) bool {
	if v == nil || v.Kind() != document.Object {
		return false
	}
	if checkKeys(v, a.collectionKeys([]string{"type", "nullable", "additionalProperties"}), pointer) != nil {
		return false
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil || kind != "" && kind != "object" {
		return false
	}
	value := get(v, "additionalProperties")
	if value == nil || value.Kind() != document.Object || has(value, "$ref") {
		return false
	}
	if checkKeys(value, append([]string{"type", "nullable", "format"}, primitiveBoundKeys...), pointer+"/additionalProperties") != nil {
		return false
	}
	kind, _, err = schemaType(value, pointer+"/additionalProperties")
	format := text(get(value, "format"))
	return err == nil && (kind == "string" && (format == "" || format == "uuid" || format == "date" || format == "date-time") || kind == "boolean" && format == "" ||
		kind == "integer" && (format == "" || format == "int32" || format == "int64"))
}
