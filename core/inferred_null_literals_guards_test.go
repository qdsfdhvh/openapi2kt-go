package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInferredNullLiteralsPreserveExcludedGrammarAndOriginalFailures(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"enum":[null,null]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"enum":[null,"text"]}`, "E_SCHEMA_INVALID", "enum/0"},
		{`{"enum":[false,null]}`, "E_SCHEMA_INVALID", "enum/0"},
		{`{"enum":[]}`, "E_SCHEMA_INVALID", "enum"},
		{`{"enum":null}`, "E_SCHEMA_INVALID", "enum"},
		{`{"const":null,"default":null}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"const":null,"format":""}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"const":null,"x-native-type":"custom.Value"}`, "E_SCHEMA_UNSUPPORTED", "const"},
		{`{"enum":[null],"minLength":0}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"const":null,"minLength":0}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"const":null,"nullable":"true"}`, "E_SCHEMA_INVALID", "nullable"},
		{`{"enum":[null],"description":false}`, "E_SCHEMA_INVALID", "description"},
		{`{"type":["null"],"enum":[null]}`, "E_SCHEMA_UNSUPPORTED", "type"},
		{`{"oneOf":[{"type":"string"},{"enum":[null]}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/enum"},
		{`{"oneOf":[{"type":"string"},{"const":null}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/const"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil {
				t.Fatal("Excluded grammar was silently accepted", tc, validation)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != "/components/schemas/Record/properties/value/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Original diagnostic or failure report changed", tc, validation, d.Code, d.Pointer, err)
			}
		}
	}
}

func TestInferredNullLiteralsKeepSourceIndicesAndOriginalDiscardedFields(t *testing.T) {
	for _, tc := range []struct{ input, code, pointer string }{
		{`{"Record":{"allOf":[{"properties":{"value":{"enum":[null,null]}}},{"properties":{"value":{"type":"string"}}}]}}`, "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/0/properties/value/enum/1"},
		{`{"Record":{"properties":{"value":{"type":"string"}},"additionalProperties":{"enum":[null,null]}}}`, "E_SCHEMA_INVALID", "/components/schemas/Record/additionalProperties/enum/1"},
		{`{"Value":{"const":null,"nullable":true}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value"},
		{`{"Value":{"enum":[null],"nullable":true}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value"},
		{`{"Value":{"enum":[` + strings.Repeat("null,", 239) + `null]}}`, "E_SCHEMA_INVALID", "/components/schemas/Value/enum/1"},
		{`{"Value":{"enum":[` + strings.Repeat("null,", 240) + `null]}}`, "E_ENUM_LIMIT", "/components/schemas/Value/enum"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(tc.input), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil {
				t.Fatal("Original source error was erased", tc, validation)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestInferredNullLiteralsKeepNullableStringVocabularyAndNullTokenDistinct(t *testing.T) {
	input := schemas(`{"Record":{"properties":{"mixed":{"enum":["ready",null],"nullable":true},"word":{"const":"null"},"literal":{"const":null},"value":{"enum":[null],"nullable":true}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "RecordDto")
	for _, marker := range []string{"public val literal: kotlinx.serialization.json.JsonNull", `public val word: kotlin.String = "null"`, "public val value: kotlinx.serialization.json.JsonNull? = null", "public sealed class Mixed", `"ready"`} {
		if !strings.Contains(text, marker) {
			t.Fatal("Null inference erased String vocabulary or token identity", marker, text)
		}
	}
}
