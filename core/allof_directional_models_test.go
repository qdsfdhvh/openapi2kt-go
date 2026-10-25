package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestOrdinaryAllOfDirectionalRequestResponseViews(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
	var doc map[string]any
	if e := json.Unmarshal(input, &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["Account"] = map[string]any{"allOf": []any{catalog["Account"]}}
	input, e := json.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal("ordinaryallOf directional DTO rejected", e)
		}
		req, res := source(t, r, "SaveAccountRequestDto"), source(t, r, "SaveAccountPublicResponseDto")
		if !strings.Contains(req, "id: org.example.models.models.Patch<kotlin.String>") || !strings.Contains(req, "password: kotlin.String,") || strings.Contains(res, "password") || !strings.Contains(res, "id: kotlin.String,") {
			t.Fatal("allOfdirectional fields leaked/requiredlost", req, res)
		}
	}
}

func directionalAccountSchema(t *testing.T, schema string) []byte {
	t.Helper()
	var root map[string]any
	if e := json.Unmarshal(directionalDoc(`"id":{"type":"string"}`), &root); e != nil {
		t.Fatal(e)
	}
	var account any
	if e := json.Unmarshal([]byte(schema), &account); e != nil {
		t.Fatal(e)
	}
	root["components"].(map[string]any)["schemas"].(map[string]any)["Account"] = account
	input, e := json.Marshal(root)
	if e != nil {
		t.Fatal(e)
	}
	return input
}

func TestAllOfDirectionalWinningAnnotationsAndMergedRequired(t *testing.T) {
	input := directionalAccountSchema(t, `{"allOf":[{"properties":{"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true}},"required":["id","password"]},{"properties":{"id":{"type":"string","readOnly":false},"password":{"type":"string","writeOnly":false,"default":"fallback"},"server":{"type":"string","readOnly":true},"secret":{"type":"string","writeOnly":true}}}]}`)
	r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	req, res := source(t, r, "SaveAccountRequestDto"), source(t, r, "SaveAccountPublicResponseDto")
	if !strings.Contains(req, "public val id: kotlin.String,") || !strings.Contains(req, "public val password: kotlin.String,") || !strings.Contains(req, "server: org.example.models.models.Patch<kotlin.String>") || strings.Contains(req, "fallback") {
		t.Fatal("winningannotation/mergedrequired lost in request", req)
	}
	if !strings.Contains(res, "public val password: kotlin.String,") || strings.Contains(res, "secret") {
		t.Fatal("discardedwriteOnly permission leaked", res)
	}
}

func TestAllOfDirectionalStillValidatesDiscardedAndHiddenSources(t *testing.T) {
	for _, tc := range []struct{ schema, code, pointer string }{
		{`{"allOf":[{"properties":{"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":"bad"}}},{"properties":{"password":{"type":"string"}}}]}`, "E_SCHEMA_INVALID", "/components/schemas/Account/allOf/0/properties/password/writeOnly"},
		{`{"allOf":[{"properties":{"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true,"format":"unsupported"}}}]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Account/allOf/0/properties/password/format"},
	} {
		failure(t, directionalAccountSchema(t, tc.schema), tc.code, tc.pointer)
	}
	input := directionalAccountSchema(t, `{"minProperties":1,"allOf":[{"properties":{"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true}}}]}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	if e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("directional allOf counts silently weakened", e)
	}
}

func TestAllOfDirectionalNamedContributionsAndSlotLimits(t *testing.T) {
	input := directionalAccountSchema(t, `{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"password":{"type":"string","writeOnly":true}},"required":["password"]}]}`)
	var root map[string]any
	if e := json.Unmarshal(input, &root); e != nil {
		t.Fatal(e)
	}
	catalog := root["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["Base"] = map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string", "readOnly": true}}, "required": []string{"id"}}
	input, e := json.Marshal(root)
	if e != nil {
		t.Fatal(e)
	}
	r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, r, "SaveAccountRequestDto"), "password: kotlin.String,") || strings.Contains(source(t, r, "SaveAccountPublicResponseDto"), "password") {
		t.Fatal("namedallOf source projection lost annotations/required")
	}
	for _, width := range []int{238, 239} {
		properties := map[string]any{"password": map[string]any{"type": "string", "writeOnly": true}}
		for i := range width {
			properties[fmt.Sprint("p", i)] = map[string]any{"type": "string"}
		}
		catalog["Account"] = map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/Base"}, map[string]any{"properties": properties, "required": []string{"password"}}}}
		input, e := json.Marshal(root)
		if e != nil {
			t.Fatal(e)
		}
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 238 && e != nil {
			t.Fatal("directional240 rejected", e)
		}
		if width == 239 && (e == nil || len(r.Artifacts.Files) != 0) {
			t.Fatal("directional241 accepted", e)
		}
	}
}
