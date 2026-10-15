package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func commonDictionaryDocument(left, right, extra string, optional bool) []byte {
	required := `,"required":["value"]`
	if optional {
		required = ""
	}
	return schemas(`{"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":` + left + `},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":` + right + `}` + required + `}]}` + extra + `}`)
}

func TestUnionCommonDictionariesTypesPresenceAndDistinctChecks(t *testing.T) {
	for _, tc := range []struct {
		name, left, right, extra, want string
		optional                       bool
	}{
		{"different-values", `{"additionalProperties":{"type":"string"}}`, `{"additionalProperties":{"type":"integer"}}`, "", "", false},
		{"optional", `{"additionalProperties":{"type":"string"}}`, `{"additionalProperties":{"type":"string"}}`, "", "kotlin.collections.Map<kotlin.String, kotlin.String>?", true},
		{"nullable-value", `{"additionalProperties":{"type":["string","null"]}}`, `{"additionalProperties":{"type":["string","null"]}}`, "", "kotlin.collections.Map<kotlin.String, kotlin.String?>", false},
		{"different-value-null", `{"additionalProperties":{"type":"string"}}`, `{"additionalProperties":{"type":["string","null"]}}`, "", "", false},
		{"json-object", `{"additionalProperties":true}`, `{"additionalProperties":{}}`, "", "kotlinx.serialization.json.JsonObject", false},
		{"named-dto", `{"additionalProperties":{"$ref":"#/components/schemas/Value"}}`, `{"additionalProperties":{"$ref":"#/components/schemas/Value"}}`, `,"Value":{"properties":{"id":{"type":"string"}},"required":["id"]}`, "kotlin.collections.Map<kotlin.String, org.example.models.models.ValueDto>", false},
		{"named-enum", `{"additionalProperties":{"$ref":"#/components/schemas/Value"}}`, `{"additionalProperties":{"$ref":"#/components/schemas/Value"}}`, `,"Value":{"enum":["x","y"]}`, "kotlin.collections.Map<kotlin.String, org.example.models.models.ValueDto>", false},
		{"aliased-container", `{"additionalProperties":{"$ref":"#/components/schemas/Rows"}}`, `{"additionalProperties":{"type":"array","items":{"type":"string"}}}`, `,"Rows":{"type":"array","items":{"type":"string"}}`, "kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.String>>", false},
		{"distinct-bounds", `{"additionalProperties":{"type":"string","minLength":2},"minProperties":1}`, `{"additionalProperties":{"type":"string","minLength":3},"maxProperties":2}`, "", "kotlin.collections.Map<kotlin.String, kotlin.String>", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				rq := request(commonDictionaryDocument(tc.left, tc.right, tc.extra, tc.optional), "Dto")
				rq.Options.Validation = checked
				result, err := core.GenerateWithDiagnostics(rq)
				if tc.name == "distinct-bounds" && !checked {
					if err == nil || len(result.Artifacts.Files) != 0 {
						t.Fatal("validation-off bounds must retain rejection", result, err)
					}
					continue
				}
				if err != nil {
					t.Fatal(checked, err)
				}
				text := source(t, result, "ChoiceDto")
				if tc.want == "" {
					if strings.Contains(text, "abstract val value:") {
						t.Fatal(text)
					}
				} else if !strings.Contains(text, "abstract val value: "+tc.want) {
					t.Fatal(text)
				}
				if tc.name == "distinct-bounds" && checked {
					for _, s := range []string{"require(length >= 2)", "require(length >= 3)", "value.size >= 1", "value.size <= 2"} {
						if !strings.Contains(text, s) {
							t.Fatal("lost branch validation", s, text)
						}
					}
				}
			}
		})
	}
}

func TestUnionCommonDictionariesOriginalSourceNoDropAndGrammar(t *testing.T) {
	field := `{"additionalProperties":{"type":"string"}}`
	for _, bad := range []string{`{"additionalProperties":{"type":"string","format":"unsupported"}}`, `{"additionalProperties":{"enum":["x","y"]}}`, `{"additionalProperties":{"type":"string"},"required":["undeclared"]}`, `{"$ref":"#/components/schemas/Mapping","unexpected":true}`} {
		rq := request(commonDictionaryDocument(bad, field, `,"Mapping":`+field, false), "Dto")
		result, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(result.Artifacts.Files) != 0 {
			t.Fatal(bad, result, err)
		}
	}
	// A later allOf replacement cannot erase an original dictionary leaf check.
	input := schemas(`{"Choice":{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"additionalProperties":{"type":"string","minLength":2}}}},{"properties":{"kind":{"const":"b"},"value":` + field + `}}]}],"properties":{"value":` + field + `}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	result, err := core.GenerateWithDiagnostics(rq)
	if err == nil || len(result.Artifacts.Files) != 0 {
		t.Fatal(result, err)
	}
}
