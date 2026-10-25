package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstFieldsKeepGrammarAndScopeBoundaries(t *testing.T) {
	for _, tc := range []struct{ value, code, tail string }{
		{`{"type":"object","const":[]}`, "E_SCHEMA_INVALID", "const"},
		{`{"type":"array","const":{}}`, "E_SCHEMA_INVALID", "const"},
		{`{"type":"string","const":{}}`, "E_SCHEMA_INVALID", "const"},
		{`{"const":{},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":"object","const":{},"format":""}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"const":[],"x-native-type":"custom.Value"}`, "E_SCHEMA_UNSUPPORTED", "x-native-type"},
		{`{"const":{},"x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "x-kotlin-value-class"},
		{`{"const":{},"properties":{}}`, "E_SCHEMA_UNSUPPORTED", "properties"},
		{`{"const":[],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "items"},
		{`{"const":{},"additionalProperties":false}`, "E_SCHEMA_UNSUPPORTED", "additionalProperties"},
		{`{"const":{},"minProperties":0}`, "E_SCHEMA_UNSUPPORTED", "minProperties"},
		{`{"const":[],"minItems":0}`, "E_SCHEMA_UNSUPPORTED", "minItems"},
		{`{"const":{},"nullable":"bad"}`, "E_SCHEMA_INVALID", "nullable"},
		{`{"const":{},"title":false}`, "E_SCHEMA_INVALID", "title"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.value), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil {
				t.Fatal("Structured const erased excluded grammar", tc, validation)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != "/components/schemas/Record/properties/value/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				if d.Pointer != nil {
					t.Log("Actual pointer:", *d.Pointer)
				}
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestStructuredConstFieldsDoNotAuthorizeUnionPinsOrDiscardedLiterals(t *testing.T) {
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{}}}},{"properties":{"tag":{"const":"two"}}}]}}`,
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if strings.Contains(input, `"Choice":`) {
				assertPinnedStructuredConstGraduation(t, got, err)
				continue
			}
			if strings.Contains(input, `"additionalProperties":{"const":`) {
				assertDiscardedStructuredConstGraduation(t, got, err)
				continue
			}
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Ordinary body permission leaked to another role", input, validation, err)
			}
		}
	}
}

func TestStructuredConstDiscardedSourcesStayValidatedAndDoNotLeakWarnings(t *testing.T) {
	valid := schemas(`{"Holder":{"allOf":[{"properties":{"value":{"const":{"type":"data","enum":[null]}}}},{"properties":{"value":{"type":"string"}}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(valid, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, warning := range got.Warnings {
		if warning.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
			t.Fatal("Discarded scratch const leaked a warning", got.Warnings)
		}
	}
	for _, tc := range []struct{ input, pointer string }{
		{`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","const":{}}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Holder/allOf/0/properties/value/const"},
		{`{"Holder":{"allOf":[{"properties":{"value":{"const":{},"default":{}}}},{"properties":{"value":{"type":"string"}}}]}}`, "/components/schemas/Holder/allOf/0/properties/value/default"},
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(tc.input), "Dto"))
		if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("An original discarded structured source was erased", tc, err)
		}
	}
}

func TestStructuredConstFieldsStayNonNullableBodiesAndRequestDefaults(t *testing.T) {
	for _, method := range []string{"post", "patch"} {
		input := []byte(`{"openapi":"3.1.0","paths":{"/holder":{"` + method + `":{"operationId":"send","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"value":{"type":"string"},"fixed":{"const":{"n":1},"nullable":true}},"required":["fixed"]}}}},"responses":{"204":{}}}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil {
			t.Fatal(method, err)
		}
		text := source(t, got, "SendInputDto")
		if !strings.Contains(text, "public val fixed: kotlinx.serialization.json.JsonObject =") || strings.Contains(text, "Patch<kotlinx.serialization.json.JsonObject>") || strings.Contains(text, "fixed: kotlinx.serialization.json.JsonObject?") {
			t.Fatal("Const body gained nullable/Patch or constructor API", method, text)
		}
	}
}
