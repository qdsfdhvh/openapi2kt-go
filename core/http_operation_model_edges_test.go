package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPOperationModelInheritanceAndOwnership(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/rows":{"parameters":[{"name":"state","in":"query","schema":{"type":"array","items":{"type":"string","enum":["on","off"]}}}],"get":{"operationId":"one","responses":{"204":{}}},"post":{"operationId":"two","responses":{"204":{}}}}}}`)
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, name := range []string{"OneStateDto", "TwoStateDto"} {
		if !strings.Contains(api, "org.example.models.api.models."+name) {
			t.Fatal(api)
		}
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
	var unordered any
	if err := json.Unmarshal(input, &unordered); err != nil {
		t.Fatal(err)
	}
	reordered, err := json.Marshal(unordered)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, report(t, reordered, "Dto")) {
		t.Fatal("operation model order changed output")
	}
}

func TestHTTPFallbackRenamingAlsoRenamesModels(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{
		"/rows":{"get":{"parameters":[{"name":"x","in":"query","required":true,"schema":{"type":"string"}},{"name":"mode","in":"query","schema":{"type":"array","items":{"type":"string","enum":["a","b"]}}}],"responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}},
		"/rows/{x}":{"get":{"parameters":[{"name":"x","in":"path","required":true,"schema":{"type":"string"}},{"name":"mode","in":"query","schema":{"type":"array","items":{"type":"string","enum":["a","b"]}}}],"responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}}
	}}`)
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, name := range []string{"GetRowsModeDto", "GetRows2ModeDto", "GetRowsResponseDto", "GetRows2ResponseDto"} {
		_ = source(t, r, name)
		if !strings.Contains(api, "org.example.models.api.models."+name) {
			t.Fatal(api)
		}
	}
}

func TestHTTPOnlySelectedResponseOwnsModels(t *testing.T) {
	r := report(t, httpDoc(`{"operationId":"chosen","responses":{"201":{"content":{"application/json":{"schema":{"type":"object","properties":{"ignored":{"type":"boolean"}}}}}},"200":{"content":{"application/other+json":{"schema":{"type":"object","properties":{"other":{"type":"boolean"}}}},"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}`), "Dto")
	model := source(t, r, "ChosenResponseDto")
	if len(r.Artifacts.Files) != 3 || strings.Contains(model, "ignored") || strings.Contains(model, "other") || !strings.Contains(model, "id:") {
		t.Fatal(r, model)
	}
}

func TestHTTPIdenticalLocalIdentityIsShared(t *testing.T) {
	responses := json.RawMessage(`{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}}}}}}}`)
	op := func(tag string) any {
		return map[string]any{"operationId": "shared", "tags": []string{tag}, "responses": responses}
	}
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/a": map[string]any{"get": op("A")}, "/b": map[string]any{"get": op("B")}}})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 4 {
		t.Fatal(r)
	}
	_ = source(t, r, "SharedResponseDto")
}

func TestHTTPOperationModelNullableAndGuards(t *testing.T) {
	r := report(t, httpDoc(`{"operationId":"maybe","responses":{"200":{"content":{"application/json":{"schema":{"type":["array","null"],"items":{"type":["object","null"],"properties":{"id":{"type":"string"}}}}}}}}}`), "Dto")
	api := source(t, r, "DefaultApi")
	if !strings.Contains(api, "kotlin.collections.List<org.example.models.api.models.MaybeItemDto?>?") {
		t.Fatal(api)
	}
	// A different pointer is not permission to overwrite an operation-local file.
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"operationId":"same","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"a":{"type":"string"}}}}}}}}},"/b":{"get":{"operationId":"same","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"b":{"type":"string"}}}}}}}}}}}`)
	failure(t, input, "E_NAME_COLLISION", "/paths/~1b/get")
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"array","uniqueItems":true,"items":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/responses/200/content/application~1json/schema/uniqueItems")
}
