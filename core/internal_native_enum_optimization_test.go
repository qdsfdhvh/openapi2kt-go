package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalNativeEnumPointerVocabularyAndBoundsRemainReuseIdentity(t *testing.T) {
	r := request(schemas(`{"Source":{"properties":{"a":{"type":"integer","enum":[1,2,3],"minimum":2},"b":{"type":"integer","enum":[1,2,3],"minimum":3}}},"Alpha":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/a"}}},"Beta":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/a"}}},"Gamma":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/b"}}}}`), "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("borrowed constraints erased")
	}
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(got, again) {
		t.Fatal(e)
	}
}

func TestInternalNativeEnumPointerChecksCannotDisappearInRawOrDiscardedRoles(t *testing.T) {
	for _, input := range [][]byte{
		schemas(`{"Source":{"properties":{"leaf":{"type":"integer","enum":[1,2]}}},"Discarded":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/leaf"}}},{"properties":{"value":{"type":"integer"}}}]}}`),
		schemas(`{"Source":{"properties":{"leaf":{"type":"integer","enum":[1,2]}}},"Ignored":{"properties":{"id":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Source/properties/leaf"}}}`),
		[]byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":"integer","enum":[1,2]}}}}},"paths":{"/raw":{"get":{"parameters":[{"name":"value","in":"query","schema":{"properties":{"code":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}],"responses":{"204":{}}}}}}`),
	} {
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/enum") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(e)
		}
	}
}
