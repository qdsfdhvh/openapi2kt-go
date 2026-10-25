package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// inferredNullLiteral reads the original literal without adding a type node or
// inferring nullable intent. Mixed vocabularies and native/format profiles retain
// their existing planners; every accepted source still passes its full grammar.
func inferredNullLiteral(v *document.Value) bool {
	if has(v, "type") || has(v, "format") || has(v, "x-native-type") || has(v, "x-kotlin-value-class") {
		return false
	}
	if value, exists := v.Get("const"); exists {
		return value.Kind() == document.Null
	}
	values := get(v, "enum")
	if values.Kind() != document.Array || len(values.Items()) == 0 {
		return false
	}
	for _, value := range values.Items() {
		if value.Kind() != document.Null {
			return false
		}
	}
	return true
}

// Null-only vocabularies reuse the existing JsonNull carrier. They do not infer
// nullability or introduce an enum declaration or a new serializer.
func nullOnlyEnumType(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	return nullOnlyEnumProfile(v, pointer, false)
}

// Explicitly scoped own-default planning admits the annotation: direct DTO
// fields, shared default walks and marked component roots. Other typed roles
// continue to use the strict profile above.
func nullOnlyEnumProfile(v *document.Value, pointer string, fieldDefault bool) (string, *diagnostic.Diagnostic) {
	location := naming.Pointer(pointer, "enum")
	if text(get(v, "type")) != "null" && !inferredNullLiteral(v) {
		return "", unsupported(location)
	}
	keys := []string{"type", "nullable", "enum"}
	if fieldDefault {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return "", err
	}
	if _, _, err := schemaType(v, pointer); err != nil {
		return "", err
	}
	values := get(v, "enum")
	if values.Kind() != document.Array || len(values.Items()) == 0 {
		return "", invalid("enum must be a non-empty array of null literals.", location)
	}
	if len(values.Items()) > MaxMembers {
		return "", diagnostic.At("E_ENUM_LIMIT", "Enum exceeds the supported 240-member limit.", location)
	}
	for i, value := range values.Items() {
		p := naming.Pointer(location, strconv.Itoa(i))
		if value.Kind() != document.Null {
			return "", invalid("A null-only enum member must be JSON null.", p)
		}
		if i > 0 {
			return "", invalid("Null enum literal entries must be unique.", p)
		}
	}
	return scalarMapping("null", v, pointer)
}
