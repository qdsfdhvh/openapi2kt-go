package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func presenceDocument(t *testing.T, body string, components any) []byte {
	t.Helper()
	v := map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/apply": map[string]any{"post": map[string]any{"requestBody": json.RawMessage(body), "responses": map[string]any{"204": map[string]any{}}}}}}
	if components != nil {
		v["components"] = components
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

const presenceBody = `{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"memo":{"type":["string","null"]}}}}}}`

func TestHTTPRequestReferenceNullableContext(t *testing.T) {
	body := `{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["mandatory"],"properties":{"inherited":{"$ref":"#/components/schemas/Maybe"},"disabled":{"$ref":"#/components/schemas/Maybe","nullable":false},"mandatory":{"$ref":"#/components/schemas/Maybe"}}}}}}`
	components := map[string]any{"schemas": map[string]any{"Maybe": map[string]any{"type": "object", "nullable": true, "additionalProperties": map[string]any{"type": "string"}}}}
	r := report(t, presenceDocument(t, body, components), "Dto")
	model := source(t, r, "PostApplyInputDto")
	for _, want := range []string{"inherited: org.example.models.models.Patch<org.example.models.models.MaybeDto>", "disabled: org.example.models.models.MaybeDto? = null", "mandatory: org.example.models.models.MaybeDto?,"} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q in %s", want, model)
		}
	}
}

func TestHTTPRequestPatchSlots(t *testing.T) {
	for _, count := range []int{240, 241} {
		properties := map[string]any{}
		for i := 0; i < count; i++ {
			properties[fmt.Sprintf("p%d", i)] = map[string]any{"type": []string{"integer", "null"}, "format": "int64"}
		}
		body, err := json.Marshal(map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": properties}}}})
		if err != nil {
			t.Fatal(err)
		}
		input := presenceDocument(t, string(body), nil)
		if count == 240 {
			report(t, input, "")
		} else {
			failure(t, input, "E_MODEL_LIMIT", "/paths/~1apply/post/requestBody/content/application~1json/schema")
		}
	}
}

func TestHTTPRequestRuntimeOwnershipAndNames(t *testing.T) {
	for _, name := range []string{"Patch", "PatchSerializer"} {
		input := presenceDocument(t, presenceBody, map[string]any{"schemas": map[string]any{name: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}}})
		failure(t, input, "E_NAME_COLLISION", "/components/schemas/"+name)
		// A suffix separates the user type from the fixed runtime identity.
		r := report(t, input, "Dto")
		_ = source(t, r, "Patch")
		_ = source(t, r, name+"Dto")
		rq := request(input, "")
		rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil || len(r.Artifacts.Files) != 2 {
			t.Fatal(r, err)
		}
	}
	// A non-selected representation cannot force a runtime into the output.
	input := presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"string"}},"application/other+json":{"schema":{"type":"object","properties":{"memo":{"type":["string","null"]}}}}}}`, nil)
	r := report(t, input, "")
	if len(r.Artifacts.Files) != 2 {
		t.Fatal(r)
	}
	// Dictionary keys already represent absence; nullable values are not Patch.
	r = report(t, presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"object","additionalProperties":{"type":["string","null"]}}}}}`, nil), "Dto")
	alias := source(t, r, "PostApplyInputDto")
	if len(r.Artifacts.Files) != 3 || strings.Contains(alias, "Patch") || !strings.Contains(alias, "kotlin.String?") {
		t.Fatal(r, alias)
	}
}
