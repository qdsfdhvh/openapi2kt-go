package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Raw JSON carrier field own defaults. A field whose shared plan produces a
// JsonElement (unconstrained/empty object or request-context union projection)
// or an open JsonObject dictionary may carry its own `default`. The value is
// embedded as a literal and parsed by a file-level helper, so no constructor
// qualifier can be shadowed.
func rawJSONDefaultCandidate(v *document.Value) bool {
	if v == nil || !has(v, "default") {
		return false
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "format", "items", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	if dictionarySchema(v) {
		flag, ok := get(v, "additionalProperties").Bool()
		return ok && flag
	}
	if properties, ok := v.Get("properties"); ok {
		if properties.Kind() != document.Object {
			return false
		}
		if len(properties.Members()) > 0 {
			return false
		}
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	return kind == "" || kind == "object"
}

func (a *analyzer) rawJSONDefaultField(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	validation := *a
	validation.warnings = nil
	if _, err := validation.rawJSONDefaultEncoded(f, ty); err != nil {
		return TypeRef{}, err
	}
	return ty, nil
}

func (a *analyzer) rawJSONDefaultInitializer(f field) (string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	ty, _, err := a.modelPropertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth)
	if err != nil {
		return "", err
	}
	return a.rawJSONDefaultEncoded(f, ty)
}

func (a *analyzer) rawJSONDefaultEncoded(f field, ty TypeRef) (string, *diagnostic.Diagnostic) {
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	carrier := ty.WithoutRootNull()
	marker := ""
	switch carrier.Kind {
	case JSON:
		marker = "`openapi-kt json default`("
	case JSONObject:
		marker = "`openapi-kt object default`("
	default:
		return "", unsupported(at)
	}
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	if value.Kind() == document.Null {
		// ty is the shared schema plan before optional presence adds a nullable
		// Kotlin use. It also retains null branches of request union projections.
		if ty.Kind != Nullable {
			return "", invalid("A null default requires a nullable schema.", at)
		}
		return "null", nil
	}
	if marker == "`openapi-kt object default`(" && value.Kind() != document.Object {
		return "", invalid("A raw object default must be a JSON object.", at)
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return "", invalid("A raw JSON default must be valid JSON.", at)
	}
	return marker + naming.Literal(string(encoded)) + ")", nil
}
