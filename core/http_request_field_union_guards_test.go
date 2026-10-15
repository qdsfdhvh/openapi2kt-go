package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func requestFieldDocument(t *testing.T, field string) []byte {
	t.Helper()
	return presenceDocument(t, `{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"payload":`+field+`}}}}}`, nil)
}

func TestHTTPRequestFieldUnionGuards(t *testing.T) {
	base := "/paths/~1apply/post/requestBody/content/application~1json/schema/properties/payload"
	for _, tc := range []struct{ field, code, tail string }{
		{`{"oneOf":[{"type":"boolean"},{"type":"array"}]}`, "E_SCHEMA_INVALID", "/oneOf/1"},
		{`{"oneOf":[{"type":"boolean"},{"type":"array","items":false}]}`, "E_SCHEMA_INVALID", "/oneOf/1/items"},
		{`{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}`, "E_SCHEMA_UNSUPPORTED", "/anyOf/0/pattern"},
		{`{"oneOf":[{"type":"string","enum":[false]},{"type":"boolean"}]}`, "E_SCHEMA_INVALID", "/oneOf/0/enum/0"},
		{`{"oneOf":[{"$ref":"#/components/schemas/Missing"},{"type":"boolean"}]}`, "E_REF_MISSING", "/oneOf/0"},
		{`{"oneOf":[{"type":"boolean"},{"type":"object","properties":{"bad":{"$ref":"#/components/schemas/Missing"}}}]}`, "E_REF_MISSING", "/oneOf/1/properties/bad"},
		{`{"oneOf":[false,{"type":"string"}]}`, "E_SCHEMA_INVALID", "/oneOf/0"},
		{`{"oneOf":[{"not":{}},{"type":"string"}]}`, "E_SCHEMA_UNSUPPORTED", "/oneOf/0/not"},
		{`{"oneOf":[{"type":"boolean"},{"type":"string","description":42}]}`, "E_SCHEMA_INVALID", "/oneOf/1/description"},
	} {
		input := requestFieldDocument(t, tc.field)
		failure(t, input, tc.code, base+tc.tail)
		rq := request(input, "")
		rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
			t.Fatal("excluded invalid branch bypassed validation", r, err)
		}
	}
	for _, count := range []int{240, 241} {
		branches := make([]string, count)
		for i := range branches {
			branches[i] = `{"type":"string"}`
		}
		branches[0] = `{"type":"boolean"}`
		input := requestFieldDocument(t, `{"oneOf":[`+strings.Join(branches, ",")+`]}`)
		if count == 240 {
			r := report(t, input, "")
			if len(r.Artifacts.Files) != 3 || strings.Contains(source(t, r, "PostApplyInput"), "StringPayload") {
				t.Fatal(r)
			}
		} else {
			failure(t, input, "E_UNION_LIMIT", base+"/oneOf")
		}
	}
	deep := `{"type":"string"}`
	for i := 0; i < 50; i++ {
		deep = `{"type":"array","items":` + deep + `}`
	}
	r, err := core.GenerateWithDiagnostics(request(requestFieldDocument(t, `{"oneOf":[{"type":"boolean"},`+deep+`]}`), ""))
	diagnostics, ok := err.(core.Diagnostics)
	if !ok || len(diagnostics) != 1 || diagnostics[0].Code != "E_SCHEMA_DEPTH" || diagnostics[0].Pointer == nil || !strings.HasPrefix(*diagnostics[0].Pointer, base+"/oneOf/1/items") || len(r.Artifacts.Files) != 0 {
		t.Fatal(r, err)
	}
}

func TestHTTPRequestFieldUnionEmptyAndPinnedBranches(t *testing.T) {
	for _, field := range []string{
		`{"oneOf":[{}, {"type":"boolean"}]}`,
		`{"anyOf":[{"description":"Metadata","examples":[{"oneOf":"opaque"}]}, {"type":"boolean"}]}`,
		`{"oneOf":[{"type":"string","enum":["known"]}, {"type":"boolean"}]}`,
		`{"oneOf":[{"type":"string","const":"fixed"}, {"type":"boolean"}]}`,
	} {
		r := report(t, requestFieldDocument(t, field), "")
		if len(r.Artifacts.Files) != 3 || !strings.Contains(source(t, r, "PostApplyInput"), "payload: kotlinx.serialization.json.JsonElement? = null") {
			t.Fatal(r)
		}
	}
}

func TestHTTPRequestFieldUnionRuntimeDemand(t *testing.T) {
	field := `{"anyOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}`
	for _, name := range []string{"Patch", "PatchSerializer"} {
		input := presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"object","properties":{"payload":`+field+`}}}}}`, map[string]any{"schemas": map[string]any{name: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}}})
		failure(t, input, "E_NAME_COLLISION", "/components/schemas/"+name)
		_ = source(t, report(t, input, "Dto"), "Patch")
	}
	// Unemitted raw branch declarations cannot introduce Patch runtime demand.
	input := requestFieldDocument(t, `{"oneOf":[{"type":"boolean"},{"type":"object","properties":{"note":`+field+`}}]}`)
	if r := report(t, input, ""); len(r.Artifacts.Files) != 3 || strings.Contains(source(t, r, "PostApplyInput"), "Patch<") {
		t.Fatal(r)
	}
	input = presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"string"}},"application/other+json":{"schema":{"type":"object","properties":{"payload":`+field+`}}}}}`, nil)
	if r := report(t, input, ""); len(r.Artifacts.Files) != 2 {
		t.Fatal("unselected raw field introduced local runtime", r)
	}
}

func TestHTTPRequestFieldUnionBorrowedSlots(t *testing.T) {
	base := "/paths/~1apply/post/requestBody/content/application~1json/schema/properties/payload"
	for _, count := range []int{120, 121} {
		var fields, required []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		input := requestFieldDocument(t, `{"oneOf":[{"type":"boolean"},{"type":"object","properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}]}`)
		if count == 120 {
			r := report(t, input, "")
			if strings.Contains(source(t, r, "PostApplyInput"), "kotlin.Long") {
				t.Fatal("raw branch declaration leaked", r)
			}
		} else {
			failure(t, input, "E_MODEL_LIMIT", base+"/oneOf/1")
		}
	}
}

func TestHTTPRequestFieldUnionFallbackOwnership(t *testing.T) {
	paths := map[string]any{}
	for _, param := range []string{"a", "b"} {
		paths["/same/{"+param+"}"] = map[string]any{"post": map[string]any{
			"parameters":  []any{map[string]any{"name": param, "in": "path", "required": true, "schema": map[string]any{"type": "string"}}},
			"requestBody": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{"payload": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "boolean"}, map[string]any{"type": "null"}}}}}}}},
			"responses":   map[string]any{"204": map[string]any{}},
		}}
	}
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for _, name := range []string{"PostSameInputDto", "PostSame2InputDto"} {
		if !strings.Contains(source(t, r, name), "Patch<kotlinx.serialization.json.JsonElement>") || !strings.Contains(source(t, r, "DefaultApi"), ".api.models."+name+"? = null") {
			t.Fatal(r)
		}
	}
	if len(r.Warnings) != 2 || !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("fallback warning ownership or determinism", r.Warnings)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 1 {
		t.Fatal("excluded raw request left local model or runtime", excluded, err)
	}
}
