package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPParameterValueGuards(t *testing.T) {
	base := "/paths/~1ping/get/parameters/0/schema"
	for _, tc := range []struct{ shape, code, tail string }{
		{`{"type":"array"}`, "E_SCHEMA_INVALID", ""},
		{`{"type":"array","items":false}`, "E_SCHEMA_INVALID", "/items"},
		{`{"type":"array","items":{"type":"string","pattern":"x"}}`, "E_SCHEMA_UNSUPPORTED", "/items/pattern"},
		{`{"type":"object","additionalProperties":{"type":"array"}}`, "E_SCHEMA_INVALID", "/additionalProperties"},
		{`{"type":"object","properties":{"id":{"$ref":"#/components/schemas/Missing"}}}`, "E_REF_MISSING", "/properties/id"},
		{`{"oneOf":[{"type":"boolean"},{"type":"string","enum":[false]}]}`, "E_SCHEMA_INVALID", "/oneOf/1/enum/0"},
		{`{"oneOf":[false,{"type":"string"}]}`, "E_SCHEMA_INVALID", "/oneOf/0"},
	} {
		input := httpDoc(`{"parameters":[{"name":"value","in":"query","schema":` + tc.shape + `}],"responses":{"204":{}}}`)
		failure(t, input, tc.code, base+tc.tail)
		rq := request(input, "Dto")
		rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
			t.Fatal("excluded parameter bypassed validation", r, err)
		}
	}
	// This schema $ref must not invent a local Filter for a referenced object.
	input := []byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":{"parameters":[{"name":"value","in":"query","style":"deepObject","schema":{"$ref":"#/components/schemas/Record"}}],"responses":{"204":{}}}}},"components":{"schemas":{"Record":{"type":"object","properties":{"id":{"type":"string"}}}}}}`)
	failure(t, input, "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/style")
}

func TestHTTPNullableInlineParameterEnumBinding(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/same":{"parameters":[{"name":"states","in":"query","schema":{"type":"array","items":{"type":["string","null"],"enum":["ready"]}}}],"get":{"responses":{"204":{}}},"post":{"responses":{"204":{}}}}}}`)
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, name := range []string{"GetSameStatesDto", "PostSameStatesDto"} {
		_ = source(t, r, name)
		if !strings.Contains(api, "kotlin.collections.List<org.example.models.api.models."+name+"?>? = null") {
			t.Fatal("inherited nullable inline enum lost owner/element nullability", api)
		}
	}
	if strings.Count(api, "it?.serialName") != 2 || !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("nullable enum access/determinism", r)
	}
}

func TestHTTPStructuredFilterFallbackOwnership(t *testing.T) {
	param := `{"name":"filter","in":"query","style":"deepObject","schema":{"type":"object","properties":{"opaque":{"oneOf":[{"type":"string"},{"type":"boolean"}]}}}}`
	input := httpDoc(`{"parameters":[` + param + `],"responses":{"204":{}}}`)
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 3 || len(r.Warnings) != 2 || strings.Contains(source(t, r, "GetPingFilterDto"), "StringOpaque") {
		t.Fatal(r)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 1 {
		t.Fatal("excluded structured filter left local model", excluded, err)
	}
	// Original branch pointers survive the Filter planning placeholder.
	bad := strings.Replace(param, `{"type":"string"}`, `{"type":"string","pattern":"x"}`, 1)
	failure(t, httpDoc(`{"parameters":[`+bad+`],"responses":{"204":{}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/parameters/0/schema/properties/opaque/oneOf/0/pattern")
}
