package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInlineSpecialCollectionReferenceChildOriginalGraduation(t *testing.T) {
	// Keep the former refusal inputs as exact positive controls.
	for _, body := range []string{
		`{"Target":{"type":"array","items":{"type":"string","format":"date"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":["2024-02-29"]}}}`,
		`{"Target":{"type":"array","items":{"type":"string","format":"date-time"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":["2024-02-29T00:00:00Z"]}}}`,
		`{"Target":{"type":"array","items":{"type":"string","format":"uuid"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":["123e4567-e89b-12d3-a456-426614174000"]}}}`,
		`{"Target":{"type":"array","items":{"type":"string","format":"byte"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[[1]]}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("former special collection child refusal", checked, body, err)
			}
		}
	}
}

func TestInlineSpecialCollectionReferenceChildDefaultIsIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, format, target, alias, child, invalid, tail string
	}{
		{"date", "date", `"2024-02-29"`, `"2024-03-01"`, `"2024-04-01"`, `"2023-02-29"`, "/0"},
		{"instant", "date-time", `"2024-02-29T00:00:00Z"`, `"2024-03-01T00:00:00Z"`, `"2024-04-01T00:00:00Z"`, `"2024-02-30T00:00:00Z"`, "/0"},
		{"uuid", "uuid", `"123e4567-e89b-12d3-a456-426614174000"`, `"123e4567-e89b-12d3-a456-426614174001"`, `"123e4567-e89b-12d3-a456-426614174002"`, `"not-a-uuid"`, "/0"},
		{"byte", "byte", `[1,2]`, `[3]`, `[4,5]`, `[128]`, "/0/0"},
		{"binary", "binary", `[1,2]`, `[3]`, `[4,5]`, `[-129]`, "/0/0"},
	} {
		for _, parent := range []struct{ name, schema, location string }{
			{"list", `"type":"array","items":`, "/items"},
			{"map", `"additionalProperties":`, "/additionalProperties"},
		} {
			t.Run(tc.name+"/"+parent.name, func(t *testing.T) {
				body := `{"Values":{"type":"array","items":{"type":"string","format":"` + tc.format + `"},"default":[` + tc.target + `]},` +
					`"Alias":{"$ref":"#/components/schemas/Values","default":[` + tc.alias + `]},` +
					`"Rows":{` + parent.schema + `{"$ref":"#/components/schemas/Alias","default":[` + tc.child + `]}}}`
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						rq := request(schemas(body), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						annotated, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("inline special collection child rejected", checked, style, err)
						}
						rq.Document = schemas(strings.Replace(body, `,"default":[`+tc.child+`]}}}`, `}}}`, 1))
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil || !reflect.DeepEqual(annotated, control) {
							t.Fatal("child annotation changed generated output", checked, style, err)
						}
						invalid := strings.Replace(body, `"default":[`+tc.child+`]}}}`, `"default":[`+tc.invalid+`]}}}`, 1)
						carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows"+parent.location+"/default"+tc.tail)
					}
				}
			})
		}
	}
}

func TestInlineSpecialCollectionChildDefaultNullableAndCounts(t *testing.T) {
	body := `{"Values":{"type":"array","nullable":true,"items":{"type":"string","format":"date"},"minItems":1},` +
		`"Alias":{"$ref":"#/components/schemas/Values","nullable":false},` +
		`"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nearest nullable special collection child rejected", err)
	}
	narrow := strings.Replace(body, `"nullable":true,"default":null`, `"nullable":false,"default":null`, 1)
	carrierDefaultFailure(t, schemas(narrow), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/additionalProperties/default")
	empty := strings.Replace(body, `"default":null`, `"default":[]`, 1)
	carrierDefaultFailure(t, schemas(empty), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/additionalProperties/default")
	invalid := strings.Replace(body, `"default":null`, `"default":["2023-02-29"]`, 1)
	carrierDefaultFailure(t, schemas(invalid), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/additionalProperties/default/0")
}
