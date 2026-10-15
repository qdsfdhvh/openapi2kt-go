package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstValueGrammarStaysValidatedWithNullableViews(t *testing.T) {
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"object","const":[]}`, "E_SCHEMA_INVALID", "/const"},
		{`{"type":"array","const":{}}`, "E_SCHEMA_INVALID", "/const"},
		{`{"const":{},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"const":[],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/items"},
		{`{"const":{},"properties":{}}`, "E_SCHEMA_UNSUPPORTED", "/properties"},
		{`{"const":{},"format":"json"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"const":[],"x-native-type":"custom.Value"}`, "E_SCHEMA_UNSUPPORTED", "/x-native-type"},
		{`{"const":{},"title":false}`, "E_SCHEMA_INVALID", "/title"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(`{"Value":`+tc.schema+`}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
	for _, root := range []string{`{"const":{}}`, `{"type":"array","const":[]}`} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(`{"Value":`+root+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":true}}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "AliasDto"), "?\n") {
				t.Fatal("Original nullable alias counterexample did not graduate to a nullable JSON classifier", root, validation, err)
			}
		}
	}
}

func TestStructuredConstValueWarningsKeepActualSourcesAndDiscardedGrammar(t *testing.T) {
	got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":{"const":{"literal":true}},"Holder":{"properties":{"fixed":{"allOf":[{"$ref":"#/components/schemas/Value"}]}}}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Value/const" || !strings.Contains(source(t, got, "HolderDto"), "public val fixed: kotlinx.serialization.json.JsonObject =") {
		t.Fatal("Reference body fabricated a /const source or lost default", got.Warnings)
	}
	for _, input := range []string{
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"const":{},"default":{}}}}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Holder":{"properties":{"fixed":{"type":"string"}},"additionalProperties":{"const":{"discarded":true}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{}}}},{"properties":{"tag":{"const":"two"}}}]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Structured value permission erased source or entered pinned/discarded body", input, err)
		}
	}
}
