package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nativeJSONErasureInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/native-json-erasure.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNativeJSONMethodErasureEnvelope(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nativeJSONErasureInput(t), suffix)
			req.Options.Validation = validation
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(err)
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", err)
			}
			s := source(t, r, "DefaultApi")
			for _, name := range []string{"send", "send2", "send3", "putPut", "putPut2", "putPut3"} {
				if strings.Count(s, "public suspend fun "+name+"(") != 1 {
					t.Fatal("native/builtin JSON classifier collision", name, s)
				}
			}
		}
	}
}

func TestNativeJSONMethodErasureMinimalPairs(t *testing.T) {
	for _, pair := range []struct{ native, builtin string }{
		{`{"x-native-type":"kotlinx.serialization.json.JsonElement"}`, `{"properties":{}}`},
		{`{"x-native-type":"kotlinx.serialization.json.JsonObject"}`, `{"additionalProperties":true}`},
	} {
		for _, nullable := range []bool{false, true} {
			extra := ""
			if nullable {
				extra = `,"nullable":true`
			}
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Native":` + pair.native + `,"Builtin":` + pair.builtin + `}},"paths":{"/a":{"post":{"operationId":"deliver","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Native"` + extra + `}}}},"responses":{"204":{}}}},"/b":{"post":{"operationId":"deliver","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Builtin"}}}},"responses":{"204":{}}}}}}`)
			r, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err != nil {
				t.Fatal(err)
			}
			s := source(t, r, "DefaultApi")
			if strings.Count(s, "public suspend fun deliver(") != 1 || strings.Count(s, "public suspend fun deliver2(") != 1 {
				t.Fatal("minimal pair", nullable, s)
			}
		}
	}
}

func TestNativeJSONMethodErasureReservedControl(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(nativeJSONErasureInput(t), &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	paths["/0-reserved"] = map[string]any{"post": map[string]any{"operationId": "send2", "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "string"}}}}, "responses": map[string]any{"204": map[string]any{}}}}
	b, _ := json.Marshal(doc)
	r, err := core.GenerateWithDiagnostics(request(b, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	s := source(t, r, "DefaultApi")
	for _, name := range []string{"send", "send2", "send3", "send4"} {
		if strings.Count(s, "public suspend fun "+name+"(") != 1 {
			t.Fatal("reserved numeric suffix", name, s)
		}
	}
	// Whole-envelope canonical control changes only IDs, retaining every schema.
	for route, raw := range paths {
		op := raw.(map[string]any)["post"].(map[string]any)
		op["operationId"] = "control" + strings.ReplaceAll(route, "-", "")
	}
	b, _ = json.Marshal(doc)
	for _, validation := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = validation
		if _, err := core.GenerateWithDiagnostics(req); err != nil {
			t.Fatal("whole control", err)
		}
	}
}
