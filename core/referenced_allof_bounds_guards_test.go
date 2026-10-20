package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedNestedAllOfBoundsKeepExcludedShapeAndRawRoles(t *testing.T) {
	composed := `{"type":"object","allOf":[{"type":"object","allOf":[{"properties":{"value":{"type":"string","minLength":1}}}]},{"properties":{"value":{"type":"string","maxLength":2}}}]}`
	for _, input := range [][]byte{
		schemas(`{"Choice":{"oneOf":[` + composed + `,{"properties":{"other":{"type":"boolean"}}}]}}`),
		schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Record"}],"discriminator":{"propertyName":"tag"}},"Record":{"allOf":[{"properties":{"tag":{"type":"string","enum":["x"]},"value":{"type":"string","minLength":1}}},{"type":"object","allOf":[{"properties":{"value":{"type":"string","maxLength":2}}}]}]}}`),
		httpDoc(`{"parameters":[{"name":"p","in":"query","schema":` + composed + `}],"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":` + composed + `}}},"responses":{"204":{}}}`),
		schemas(`{"Record":{"properties":{"value":{"type":"string"}},"additionalProperties":` + composed + `}}`),
	} {
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
			t.Fatal("Excluded shape/raw contribution gained aggregate privilege", string(input), e)
		}
	}
}

func TestReferencedNestedAllOfBoundsKeepSourceDiagnosticsAndNoDropEligibility(t *testing.T) {
	for _, tc := range []struct{ first, second, code, tail string }{
		{`{"type":"string","minLength":4}`, `{"type":"string","maxLength":3}`, "E_SCHEMA_INVALID", "Record/allOf/1/properties/value/maxLength"},
		{`{"type":"integer","minimum":5}`, `{"type":"integer","maximum":4}`, "E_SCHEMA_INVALID", "Record/allOf/1/properties/value/maximum"},
		{`{"type":"integer","minimum":2,"multipleOf":2}`, `{"type":"integer","maximum":3,"multipleOf":3}`, "E_SCHEMA_UNSUPPORTED", "Record/allOf/1/properties/value/multipleOf"},
		{`{"type":"string","pattern":"a"}`, `{"type":"string","pattern":"b"}`, "E_SCHEMA_UNSUPPORTED", "Record/allOf/1/properties/value/pattern"},
		{`{"type":"string","minLength":-1}`, `{"type":"string","minLength":1}`, "E_SCHEMA_INVALID", "Lower/properties/value/minLength"},
		{`{"type":"string","minLength":1}`, `{"type":["string","null"],"maxLength":4}`, "E_SCHEMA_UNSUPPORTED", "Lower/properties/value/minLength"},
		{`{"type":"integer","minimum":2}`, `{"type":"integer","enum":[2,3]}`, "E_SCHEMA_UNSUPPORTED", "Lower/properties/value/minimum"},
		{`{"type":"array","minItems":1,"items":{"type":"string"}}`, `{"type":"array","items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "Lower/properties/value/minItems"},
	} {
		r := request(schemas(`{"Lower":{"properties":{"value":`+tc.first+`}},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"properties":{"value":`+tc.second+`}}]}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != tc.code || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, e)
		}
	}
	for _, version := range []string{"3.0.3", "3.1.0"} {
		bounds := `"exclusiveMinimum":2`
		if version == "3.0.3" {
			bounds = `"minimum":2,"exclusiveMinimum":true`
		}
		r := request([]byte(`{"openapi":"`+version+`","components":{"schemas":{"Lower":{"properties":{"value":{"type":"integer",`+bounds+`}}},"Record":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"properties":{"value":{"type":"integer","maximum":4}}}]}}}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil || !strings.Contains(source(t, got, "RecordDto"), "value > 2") || !strings.Contains(source(t, got, "RecordDto"), "value <= 4") {
			t.Fatal(version, e)
		}
	}
}
