package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func scalarAliasCollectionMemberDocument(target, intent, value string, dictionary bool) []byte {
	input := nullableCollectionMemberDocument(target, intent, value, dictionary)
	text := strings.Replace(string(input), `"$ref":"#/components/schemas/Target"`, `"$ref":"#/components/schemas/Alias"`, 1)
	return []byte(strings.Replace(text, `"Rows":`, `"Alias":{"$ref":"#/components/schemas/Target"},"Rows":`, 1))
}

func TestScalarAliasCollectionObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ target, value string }{
		{`{"type":"string","format":"date"}`, `"2024-02-29"`},
		{`{"type":"string","format":"date-time"}`, `"2024-02-29T03:04:05+02:00"`},
		{`{"type":"string","format":"uuid"}`, `"550e8400-e29b-41d4-a716-446655440000"`},
		{`{"type":"string","format":"byte"}`, `[-128,0,127]`},
		{`{"type":"string","enum":["on","null"]}`, `"on"`},
		{`{"type":"integer","enum":[1,2]}`, `1`},
		{`{"type":"integer","format":"int64","enum":[1,9223372036854775807]}`, `9223372036854775807`},
		{`{"type":"number","format":"float","enum":[1,1.5]}`, `1.5`},
		{`{"type":"number","format":"double","enum":[1,1.25]}`, `1.25`},
		{`{"type":"boolean","enum":[true,false]}`, `false`},
	} {
		for _, checked := range []bool{false, true} {
			if !checked && strings.Contains(tc.target, `"format":"uuid"`) {
				continue // Named builtin aliases retain their validation-only gate.
			}
			for _, dictionary := range []bool{false, true} {
				value := `[` + tc.value + `,null]`
				if dictionary {
					value = `{"value":` + tc.value + `,"nil":null}`
				}
				input := scalarAliasCollectionMemberDocument(tc.target, `,"nullable":true`, value, dictionary)
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					baseline := r
					baseline.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
					base, err := core.GenerateWithDiagnostics(baseline)
					if err != nil {
						t.Fatal("no-default control", tc.target, checked, dictionary, err)
					}
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal("member default", tc.target, checked, dictionary, err)
					}
					for _, name := range []string{"TargetDto", "AliasDto", "RowsDto"} {
						if source(t, got, name) != source(t, base, name) {
							t.Fatal("changed declaration", name)
						}
					}
					if !reflect.DeepEqual(got.Warnings, base.Warnings) {
						t.Fatal("changed warnings")
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

func TestScalarAliasCollectionObjectDefaultBuiltinAdmission(t *testing.T) {
	for _, tc := range []struct{ target, value string }{
		{`{"type":"string","minLength":2}`, `"ab"`},
		{`{"type":"integer","minimum":2,"multipleOf":2}`, `4`},
		{`{"type":"number","format":"float","minimum":1}`, `1.5`},
		{`{"type":"boolean"}`, `true`},
	} {
		input := scalarAliasCollectionMemberDocument(tc.target, `,"nullable":true`, `[`+tc.value+`,null]`, false)
		r := request(input, "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(tc.target, err)
		}
		// Enabling a default cannot widen the reference graph's validation gate.
		r.Options.Validation = false
		for _, doc := range [][]byte{input, []byte(strings.Replace(string(input), `,"default":{"member":[`+tc.value+`,null]}`, "", 1))} {
			r.Document = doc
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("validation gate widened", tc.target, err)
			}
		}
	}
}

func TestScalarAliasCollectionObjectDefaultNullableAndValues(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, dictionary := range []bool{false, true} {
					value, tail := `[null]`, "/0"
					if dictionary {
						value, tail = `{"a/~":null}`, "/a~1~0"
					}
					input := scalarAliasCollectionMemberDocument(`{"type":["string","null"],"format":"date"}`, near, value, dictionary)
					input = []byte(strings.Replace(string(input), `"Alias":{"$ref":"#/components/schemas/Target"}`, `"Alias":{"$ref":"#/components/schemas/Target"`+far+`}`, 1))
					nullable := true
					for _, intent := range []string{far, near} {
						if intent != "" {
							nullable = intent == `,"nullable":true`
						}
					}
					if !nullable {
						carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tail)
					} else {
						r := request(input, "Dto")
						r.Options.Validation = checked
						if _, err := core.GenerateWithDiagnostics(r); err != nil {
							t.Fatal("nearest nullable", err)
						}
					}
				}
			}
		}
		for _, tc := range []struct{ target, value, tail string }{
			{`{"type":"string","format":"date"}`, `"2023-02-29"`, ""},
			{`{"type":"string","format":"date-time"}`, `"2024-01-01"`, ""},
			{`{"type":"string","format":"byte"}`, `[128]`, "/0"},
			{`{"type":"string","enum":["on"]}`, `"future"`, ""},
			{`{"type":"integer","enum":[1,2]}`, `1.0`, ""},
			{`{"type":"number","format":"float","enum":[1]}`, `1.000000001`, ""},
		} {
			carrierDefaultFailure(t, scalarAliasCollectionMemberDocument(tc.target, `,"nullable":true`, `{"a/~":`+tc.value+`}`, true), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/a~1~0"+tc.tail)
		}
	}
}

func TestScalarAliasCollectionObjectDefaultSourceGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ target, extra string }{
			{`{"type":"string","format":"date","default":"2024-02-29"}`, ""},
			{`{"type":"string","format":"date"}`, `,"default":"2024-02-29"`},
			{`{"type":"string","format":"date"}`, `,"type":"string"`},
			{`{"type":"array","items":{"type":"integer"}}`, ""},
			{`{"oneOf":[{"type":"string"},{"type":"integer"}]}`, ""},
		} {
			input := scalarAliasCollectionMemberDocument(tc.target, "", `[]`, false)
			input = []byte(strings.Replace(string(input), `"Alias":{"$ref":"#/components/schemas/Target"}`, `"Alias":{"$ref":"#/components/schemas/Target"`+tc.extra+`}`, 1))
			r := request(input, "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("source hidden by empty value", tc, err)
			}
		}
		// Bad values validate even if presence suppresses their initializer.
		definitions := `"Day":{"type":"string","format":"date"},"Alias":{"$ref":"#/components/schemas/Day"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},`
		field := `{"properties":{"member":{"$ref":"#/components/schemas/Rows"}},"nullable":true,"default":{"member":["2023-02-29"]}}`
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
	}
}
