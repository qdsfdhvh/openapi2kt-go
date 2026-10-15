package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumNullMembersKeepInternalLeavesAndNoDrop(t *testing.T) {
	input := schemas(`{"ARows":{"$ref":"#/components/schemas/Source/properties/rows"},"BMap":{"$ref":"#/components/schemas/Source/properties/map"},"ZBridge":{"$ref":"#/components/schemas/Source/properties/code"},"Consumer":{"properties":{"rows":{"$ref":"#/components/schemas/ARows"},"map":{"$ref":"#/components/schemas/BMap"}}},"Source":{"properties":{"code":{"type":["integer","null"],"enum":[null,1,2,3,4],"minimum":2},"rows":{"type":"array","items":{"$ref":"#/components/schemas/ZBridge"}},"map":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Source/properties/code"}}}}}`)
	r := request(input, "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if text := source(t, got, "ConsumerDto"); !strings.Contains(text, "value.values.forEach") || !strings.Contains(text, "value >= 2") || strings.Contains(text, "value == null") {
		t.Fatal(text)
	}
	leaf := `{"type":["integer","null"],"enum":[1,null,2]}`
	for index, input := range [][]byte{
		schemas(`{"Lost":{"allOf":[{"properties":{"value":` + leaf + `}},{"properties":{"value":{"type":"integer"}}}]}}`),
		schemas(`{"Lost":{"properties":{"id":{"type":"string"}},"additionalProperties":` + leaf + `}}`),
		httpDoc(`{"operationId":"lost","parameters":[{"name":"value","in":"query","schema":{"properties":{"value":` + leaf + `}}}],"responses":{"204":{}}}`),
	} {
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(index, "Enforcing nullable vocabulary checks dropped", e)
		}
	}
	typed := request(httpDoc(`{"operationId":"typed","responses":{"200":{"content":{"application/json":{"schema":{"type":"array","items":{"properties":{"value":`+leaf+`}}}}}}}}`), "Dto")
	typed.Options.Validation = true
	kept, e := core.GenerateWithDiagnostics(typed)
	if e != nil || !strings.Contains(source(t, kept, "TypedItemDto"), "value == 1 || value == 2") {
		t.Fatal("Response array item lost its typed DTO checks", e)
	}
}
