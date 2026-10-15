package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

const simpleDeepParam = `{"name":"filter","in":"query","style":"deepObject","schema":{"type":"object","properties":{"memo":{"type":["string","null"]}}}}`

func deepPaths(t *testing.T, paths any) []byte {
	t.Helper()
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestHTTPDeepObjectBinding(t *testing.T) {
	operation := json.RawMessage(`{"responses":{"204":{}}}`)
	input := deepPaths(t, map[string]any{"/scan": map[string]any{"parameters": []any{json.RawMessage(simpleDeepParam)}, "get": operation, "post": operation}})
	r := report(t, input, "Dto")
	for _, name := range []string{"GetScanFilterDto", "PostScanFilterDto"} {
		_ = source(t, r, name)
	}
	if len(r.Artifacts.Files) != 4 {
		t.Fatal(r)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 1 {
		t.Fatal(excluded, err)
	}

	paths := map[string]any{}
	for _, location := range []string{"path", "query"} {
		path := "/scan"
		if location == "path" {
			path += "/{id}"
		}
		paths[path] = map[string]any{"get": map[string]any{"parameters": []any{
			json.RawMessage(simpleDeepParam), map[string]any{"name": "id", "in": location, "required": true, "schema": map[string]any{"type": "string"}},
		}, "responses": map[string]any{"204": map[string]any{}}}}
	}
	r = report(t, deepPaths(t, paths), "Dto")
	for _, name := range []string{"GetScanFilterDto", "GetScan2FilterDto"} {
		_ = source(t, r, name)
	}
	if !strings.Contains(source(t, r, "DefaultApi"), "getScan2(") {
		t.Fatal(r)
	}
}

func TestHTTPDeepObjectBoundaries(t *testing.T) {
	for _, shape := range []string{`{"type":"array","items":{"type":"string","pattern":"x"}}`, `{"type":"object","properties":{"id":{"type":"string","pattern":"x"}}}`} {
		parameter := `{"name":"f","in":"query","style":"deepObject","schema":{"type":"object","properties":{"x":` + shape + `}}}`
		tail := "/items/pattern"
		if strings.Contains(shape, "properties") {
			tail = "/properties/id/pattern"
		}
		failure(t, httpDoc(`{"parameters":[`+parameter+`],"responses":{"204":{}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/parameters/0/schema/properties/x"+tail)
	}
	// Actual owner binding must validate nested names, not a placeholder name.
	parameter := `{"name":"f","in":"query","style":"deepObject","schema":{"type":"object","properties":{"getPingFilter":{"type":"string","enum":["a"]}}}}`
	failure(t, httpDoc(`{"parameters":[`+parameter+`],"responses":{"204":{}}}`), "E_NAME_COLLISION", "/paths/~1ping/get/parameters/0/schema/properties/getPingFilter")
	// Both filters own the same legacy <Operation>Filter identity: different
	// shapes cannot overwrite each other in the included output.
	second := strings.ReplaceAll(strings.ReplaceAll(simpleDeepParam, `"filter"`, `"other"`), `"memo"`, `"different"`)
	input := httpDoc(`{"parameters":[` + simpleDeepParam + `,` + second + `],"responses":{"204":{}}}`)
	failure(t, input, "E_NAME_COLLISION", "/paths/~1ping/get")
}

func TestHTTPDeepObjectNullableRoot(t *testing.T) {
	parameter := `{"name":"f","in":"query","style":"deepObject","required":true,"schema":{"type":["object","null"],"properties":{"id":{"type":"string"}}}}`
	r := report(t, httpDoc(`{"parameters":[`+parameter+`],"responses":{"204":{}}}`), "")
	if !strings.Contains(source(t, r, "DefaultApi"), "f: org.example.models.api.models.GetPingFilter,") {
		t.Fatal(r)
	}
}
