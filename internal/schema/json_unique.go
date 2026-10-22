package schema

import "openapi-kt/internal/document"

func uniqueJSONType(ty TypeRef) bool {
	if ty.Kind == Nullable {
		return uniqueJSONType(*ty.Element)
	}
	// Forward named JSON carriers may not have their backing linked yet.
	// uniqueJSONSource and final retained-declaration validation authorize them.
	if ty.Kind == Named {
		return ty.Serializer == "" && ty.Validation == nil
	}
	return (ty.Kind == JSON || ty.Kind == JSONObject || ty.Kind == Scalar && ty.Name == "kotlinx.serialization.json.JsonElement") && ty.Serializer == ""
}

// Original unconstrained JSON leaves and admitted named JSON carriers only.
// References require retained source/declaration authority; type spelling alone
// cannot grant uniqueness to a lossy projection. Complete recursive covers use
// their original graph and JSON export authority. Empty composed raw uses the
// actual empty ordinary Model authority, never its projected spelling alone.
func (a *analyzer) uniqueJSONSource(v *document.Value, pointer string, depth int) bool {
	if a.booleanTrueValue(v) {
		return true
	}
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false
	}
	if has(v, "$ref") {
		if checkKeys(v, []string{"$ref", "nullable"}, pointer) != nil {
			return false
		}
		target, err := a.resolveSchema(text(get(v, "$ref")), pointer+"/$ref")
		if err != nil {
			return false
		}
		if has(target.Schema, "$ref") {
			return a.uniqueJSONSource(target.Schema, target.Pointer, depth+1)
		}
		if a.admitBorrowedRecursiveCover(target, pointer+"/$ref") {
			return true
		}
		if has(target.Schema, "allOf") {
			return a.admitBorrowedRawTrue(target, pointer+"/$ref")
		}
		return a.uniqueJSONSource(target.Schema, target.Pointer, depth+1) && a.admitBorrowedRawTrue(target, pointer+"/$ref")
	}
	for _, key := range []string{"allOf", "oneOf", "anyOf", "enum", "const", "format", "x-native-type", "x-kotlin-value-class", "properties"} {
		if has(v, key) {
			return false
		}
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil || kind != "" && kind != "object" {
		return false
	}
	if has(v, "additionalProperties") {
		value := get(v, "additionalProperties")
		if flag, ok := value.Bool(); ok {
			return flag
		}
		return value.Kind() == document.Object && len(value.Members()) == 0
	}
	return true
}
