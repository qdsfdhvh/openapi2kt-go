package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestExplicitNullLiteralVocabularyAndExcludedGrammar(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"null","enum":[]}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"null","enum":null}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"null","enum":[null,null]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"null","enum":[null,"null"]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"null","enum":[false,null]}`, "E_SCHEMA_INVALID", "enum/0"},
		{`{"type":["null"],"enum":[null]}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"type":"null","const":"null"}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"type":"null","const":null,"format":""}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"type":"null","const":null,"x-native-type":"custom.Value"}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"type":"null","enum":[null],"format":""}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"type":"null","const":null,"default":null}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":"null","const":null,"minLength":0}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"type":"null","enum":[null],"minLength":0}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"oneOf":[{"type":"string"},{"type":"null","enum":[null]}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/enum"},
		{`{"oneOf":[{"type":"string"},{"type":"null","const":null}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/const"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(r)
			if e == nil || e.(core.Diagnostics)[0].Code != tc.code || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, e)
			}
		}
	}
	for _, literal := range []string{`{"type":"null","enum":[null]}`, `{"type":"null","const":null}`} {
		r := request(schemas(`{"Value":`+strings.TrimSuffix(literal, "}")+`,"nullable":true}}`), "Dto")
		if _, e := core.GenerateWithDiagnostics(r); e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Value" {
			t.Fatal("Nullable component root contract changed", literal, e)
		}
	}
}

func TestExplicitNullLiteralsOriginalDiscardedAndExtraGrammar(t *testing.T) {
	for _, tc := range []struct{ input, pointer string }{
		{`{"Record":{"allOf":[{"properties":{"value":{"type":"null","enum":[null,null]}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Record/allOf/0/properties/value/enum/1"},
		{`{"Record":{"allOf":[{"properties":{"value":{"type":"null","const":"null"}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Record/allOf/0/properties/value/type"},
		{`{"Record":{"properties":{"value":{"type":"string"}},"additionalProperties":{"type":"null","enum":[null,false]}}}`, "/components/schemas/Record/additionalProperties/enum/1"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(tc.input), "Dto")
			r.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(r)
			if e == nil || *e.(core.Diagnostics)[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("An original discarded/extra grammar error was erased", checked, tc.pointer, e)
			}
		}
	}
}

func TestNullOnlyEnumArtifactsMatchExistingNullScalar(t *testing.T) {
	input := schemas(`{"Value":{"type":"null","enum":[null]},"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"required":{"type":"null","enum":[null]},"borrowed":{"$ref":"#/components/schemas/Alias"},"items":{"type":"array","items":{"type":"null","enum":[null]}}},"required":["required"]}}`)
	plain := []byte(strings.ReplaceAll(string(input), `,"enum":[null]`, ""))
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(r)
		r.Document = plain
		control, err := core.GenerateWithDiagnostics(r)
		if e != nil || err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("Null-only enum changed existing carrier API/wire", checked, e, err)
		}
	}
}
