package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nullableAllOfCountsInput(root string) []byte {
	return schemas(`{"Base":{"properties":{"id":{"type":"string"}},"minProperties":1},"Record":{` + root + `,"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"note":{"type":"string"}},"maxProperties":3}],"maxProperties":2},"Alias":{"$ref":"#/components/schemas/Record"},"Narrow":{"$ref":"#/components/schemas/Alias","nullable":false}}`)
}

func TestNullableAllOfCountsOriginalCodecAndAliases(t *testing.T) {
	for _, root := range []string{`"type":["object","null"]`, `"type":"object","nullable":true`, `"nullable":true`} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(nullableAllOfCountsInput(root), "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = true, style
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(root, style, err)
			}
			text := source(t, r, "RecordDto")
			for _, token := range []string{"generatedSerializer()", "element.size < 1", "element.size > 2", "id: kotlin.String?", "note: kotlin.String?"} {
				if !strings.Contains(text, token) {
					t.Fatal(token, text)
				}
			}
			if !strings.Contains(source(t, r, "AliasDto"), "RecordDto?") || strings.Contains(source(t, r, "NarrowDto"), "RecordDto?") {
				t.Fatal("nearest nullable intent lost")
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("deterministic source/codec planning changed", err)
			}
		}
	}
}

func TestNullableAllOfCountsSourceDiagnosticsAndProjection(t *testing.T) {
	input := nullableAllOfCountsInput(`"nullable":true`)
	aliasFailure(t, []byte(strings.Replace(string(input), `"maxProperties":2`, `"maxProperties":0`, 1)), "E_SCHEMA_INVALID", "/components/schemas/Record/maxProperties")
	aliasFailure(t, []byte(strings.Replace(string(input), `"minProperties":1`, `"minProperties":-1`, 1)), "E_SCHEMA_INVALID", "/components/schemas/Base/minProperties")
	for _, own := range []string{`,"maxProperties":2`, ""} {
		counted := `{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"minProperties":1}]` + own + `}`
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":` + counted + `}},"paths":{"/filter":{"get":{"parameters":[{"name":"q","in":"query","schema":{"type":"array","items":{"$ref":"#/components/schemas/Record"}}}],"responses":{"204":{}}}}}}`)
		at := "/components/schemas/Record/allOf/0/minProperties"
		if own != "" {
			at = "/components/schemas/Record/maxProperties"
		}
		aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", at)
	}
}

func TestNullableAllOfCountsOriginalExcludedInputs(t *testing.T) {
	// Preserve the original rejected inputs as positives for this source role.
	for _, body := range []string{
		`{"nullable":true,"allOf":[{"properties":{"value":{"type":"string"}}}],"minProperties":0}`,
		`{"type":["object","null"],"properties":{"value":{"type":"string"}},"allOf":[{}],"minProperties":0}`,
		`{"nullable":true,"allOf":[{"nullable":false,"minProperties":0,"properties":{"value":{"type":"string"}}}]}`,
		`{"allOf":[{"$ref":"#/components/schemas/Target"}],"nullable":true,"minProperties":0}`,
	} {
		input := schemas(`{"Record":` + body + `,"Target":{"properties":{"value":{"type":"string"}}}}`)
		if text := source(t, aggregationReport(t, input), "Record"); !strings.Contains(text, "generatedSerializer()") {
			t.Fatal("composed count serializer missing", text)
		}
		if _, err := core.Generate(request(input, "")); err == nil {
			t.Fatal("count validation opt-in lost")
		}
	}
}
