package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nullableCollectionMemberDocument(target, edgeIntent, value string, dictionary bool) []byte {
	container := `{"type":"array","items":{"$ref":"#/components/schemas/Target"` + edgeIntent + `}}`
	if dictionary {
		container = `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"` + edgeIntent + `}}`
	}
	return schemas(`{"Target":` + target + `,"Rows":` + container + `,"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Rows"}},"default":{"member":` + value + `}}}}}`)
}

func TestObjectDefaultCollectionElementNullableOverrides(t *testing.T) {
	for _, target := range []string{
		`{"type":"string","format":"date"}`,
		`{"type":"string","format":"date-time"}`,
		`{"type":"string","format":"uuid"}`,
		`{"type":"string","format":"byte"}`,
		`{"type":"string","enum":["on","null"]}`,
		`{"type":"integer","enum":[1,2]}`,
	} {
		for _, checked := range []bool{false, true} {
			for _, sourceNull := range []bool{false, true} {
				if sourceNull && !checked && strings.Contains(target, `"type":"integer"`) {
					continue // Nullable native enums retain their validation-only gate.
				}
				leaf := target
				if sourceNull {
					leaf = strings.TrimSuffix(leaf, "}") + `,"nullable":true}`
				}
				for _, intent := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, dictionary := range []bool{false, true} {
						value, tail := `[null]`, "/0"
						if dictionary {
							value, tail = `{"a/~":null}`, "/a~1~0"
						}
						input := nullableCollectionMemberDocument(leaf, intent, value, dictionary)
						nullable := sourceNull
						if intent != "" {
							nullable = intent == `,"nullable":true`
						}
						if !nullable {
							carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tail)
							continue
						}
						for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
							r := request(input, "Dto")
							r.Options.Validation, r.Options.ModelStyle = checked, style
							got, err := core.GenerateWithDiagnostics(r)
							if err != nil {
								t.Fatal(target, sourceNull, intent, dictionary, checked, err)
							}
							baseline := r
							baseline.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
							base, err := core.GenerateWithDiagnostics(baseline)
							if err != nil || source(t, got, "RowsDto") != source(t, base, "RowsDto") || source(t, got, "TargetDto") != source(t, base, "TargetDto") || !reflect.DeepEqual(got.Warnings, base.Warnings) {
								t.Fatal("existing types or source declarations changed", err)
							}
							again, err := core.GenerateWithDiagnostics(r)
							if err != nil || !reflect.DeepEqual(got, again) {
								t.Fatal("nondeterministic", err)
							}
						}
					}
				}
			}
		}
	}
}

func TestObjectDefaultCollectionElementNullableSuppressionAndGuards(t *testing.T) {
	definitions := `"Day":{"type":["string","null"],"format":"date"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day","nullable":false}},`
	field := `{"properties":{"member":{"$ref":"#/components/schemas/Rows"}},"nullable":true,"default":{"member":[null]}}`
	for _, checked := range []bool{false, true} {
		for _, role := range []string{"required", "patch", "discarded"} {
			holder := `{"required":["v"],"properties":{"v":` + field + `}}`
			pointer := "/components/schemas/Holder/properties/v/default/member/0"
			if role == "discarded" {
				holder = `{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`
				pointer = "/components/schemas/Holder/allOf/0/properties/v/default/member/0"
			}
			input := schemas(`{` + definitions + `"Holder":` + holder + `}`)
			if role == "patch" {
				input = []byte(strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`)
				pointer = "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member/0"
			}
			carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", pointer)
		}
		// Null does not bypass original source or temporal value checks.
		carrierDefaultFailure(t, nullableCollectionMemberDocument(`{"type":"string","format":"date"}`, `,"nullable":true`, `["2023-02-29"]`, false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/0")
		carrierDefaultFailure(t, nullableCollectionMemberDocument(`{"type":"string","enum":["on"]}`, `,"nullable":true`, `{"k":"unknown"}`, true), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/k")
	}
}
