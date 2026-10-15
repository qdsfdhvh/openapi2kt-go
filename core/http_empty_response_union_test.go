package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPEmptyResponseUnions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "empty-response-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 4 {
		t.Fatal(r)
	}
	api := source(t, r, "DefaultApi")
	for _, want := range []string{
		"suspend fun emptyResult(", "suspend fun singleEmpty(", "): kotlin.Unit",
		"): org.example.models.api.models.OneObjectResponseDto?",
		"): org.example.models.models.RecordDto?", "): kotlinx.serialization.json.JsonElement",
	} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q: %s", want, api)
		}
	}
	if strings.Contains(source(t, r, "OneObjectResponseDto"), "Patch<") {
		t.Fatal("response borrowed shape gained request context")
	}
	if again := report(t, input, "Dto"); !reflect.DeepEqual(r, again) {
		t.Fatal("empty response projection is not deterministic")
	}
	warnings := 0
	for _, w := range r.Warnings {
		if w.Code == "W_HTTP_RESPONSE_PROJECTION" {
			warnings++
		}
	}
	if warnings != 11 {
		t.Fatal(r.Warnings)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 2 {
		t.Fatal(excluded, err)
	}
}

func TestHTTPEmptyResponseFallbackOwnership(t *testing.T) {
	paths := map[string]any{}
	for _, param := range []string{"a", "b"} {
		paths["/same/{"+param+"}"] = map[string]any{"get": map[string]any{
			"parameters": []any{map[string]any{"name": param, "in": "path", "required": true, "schema": map[string]any{"type": "string"}}},
			"responses":  map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"oneOf": []any{map[string]any{}, map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}}}}}}},
		}}
	}
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for _, name := range []string{"GetSameResponseDto", "GetSame2ResponseDto"} {
		if !strings.Contains(source(t, r, name), "id: kotlin.String? = null") || !strings.Contains(source(t, r, "DefaultApi"), ".api.models."+name+"?") {
			t.Fatal(r)
		}
	}
	if len(r.Warnings) != 2 {
		t.Fatal("fallback planning duplicated or lost empty-branch warnings", r.Warnings)
	}
}

func emptyResponseDocument(schema string) []byte {
	return httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":` + schema + `}}}}}`)
}

func TestHTTPEmptyResponseUnionGuards(t *testing.T) {
	base := "/paths/~1ping/get/responses/200/content/application~1json/schema"
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"anyOf":[{"not":{"type":"string"}},{"type":"string"}]}`, "E_SCHEMA_UNSUPPORTED", "/anyOf/0/not"},
		{`{"anyOf":[{"not":false},{}]}`, "E_SCHEMA_INVALID", "/anyOf/0/not"},
		{`{"anyOf":[{"description":42},{}]}`, "E_SCHEMA_INVALID", "/anyOf/0/description"},
		{`{"anyOf":[{"not":{},"unknownKeyword":true},{}]}`, "E_SCHEMA_UNSUPPORTED", "/anyOf/0/not"},
		{`{"oneOf":[{}, {"type":"object","properties":{"id":{"$ref":"#/components/schemas/Missing"}}}]}`, "E_REF_MISSING", "/oneOf/1/properties/id"},
		{`{"oneOf":[{}, {"$ref":"#/components/schemas/Missing"}]}`, "E_REF_MISSING", "/oneOf/1"},
		{`{"oneOf":[false,{}]}`, "E_SCHEMA_INVALID", "/oneOf/0"},
		{`{"not":{}}`, "E_SCHEMA_UNSUPPORTED", "/not"},
	} {
		failure(t, emptyResponseDocument(tc.schema), tc.code, base+tc.tail)
	}
	for _, count := range []int{240, 241} {
		branches := make([]string, count)
		for i := range branches {
			branches[i] = `{}`
		}
		input := emptyResponseDocument(`{"anyOf":[` + strings.Join(branches, ",") + `]}`)
		if count == 240 {
			if !strings.Contains(source(t, report(t, input, ""), "DefaultApi"), "): kotlin.Unit") {
				t.Fatal("all empty response branches must project Unit")
			}
		} else {
			failure(t, input, "E_UNION_LIMIT", base+"/anyOf")
		}
	}
	// Borrowing a selected shape preserves, rather than resets, union depth.
	item := map[string]any{"type": "string"}
	for i := 0; i < 49; i++ {
		item = map[string]any{"type": "array", "items": item}
	}
	deep, err := json.Marshal(map[string]any{"anyOf": []any{map[string]any{}, map[string]any{"type": "object", "properties": map[string]any{"values": item}}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = core.Generate(request(emptyResponseDocument(string(deep)), ""))
	if err == nil || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
		t.Fatal(err)
	}
	// A borrowed object cannot escape the existing serialized-constructor cap.
	for _, count := range []int{120, 121} {
		var fields, required []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		input := emptyResponseDocument(`{"oneOf":[{}, {"type":"object","properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}]}`)
		if count == 120 {
			report(t, input, "Dto")
		} else {
			failure(t, input, "E_MODEL_LIMIT", base+"/oneOf/1")
		}
	}
}
