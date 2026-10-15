package schema

import "openapi-kt/internal/document"

func uniqueJSONType(ty TypeRef) bool {
	if ty.Kind == Nullable {
		return uniqueJSONType(*ty.Element)
	}
	return (ty.Kind == JSON || ty.Kind == JSONObject || ty.Kind == Scalar && ty.Name == "kotlinx.serialization.json.JsonElement") && ty.Serializer == ""
}

// Original direct unconstrained JSON leaves only. Named cover/ref graph and
// nominal codecs require a separate equality authority; type spelling alone
// cannot grant uniqueness to a lossy projection.
func (a *analyzer) uniqueJSONSource(v *document.Value, pointer string, depth int) bool {
	if a.booleanTrueValue(v) {
		return true
	}
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "format", "x-native-type", "x-kotlin-value-class", "properties"} {
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
