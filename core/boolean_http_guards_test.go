package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func TestBooleanHTTPValidateDiscardedAndExcluded(t *testing.T) {
	for _, profile := range []string{"false-body", "false-response", "false-unused-media", "false-unused-status", "bad-unused-status", "false-excludedAPI", "3.0", "sse-envelope", "parameter", "unionbranch"} {
		for _, checked := range []bool{false, true} {
			var doc map[string]any
			if err := json.Unmarshal(booleanHTTPInput(t), &doc); err != nil {
				t.Fatal(err)
			}
			paths := doc["paths"].(map[string]any)
			echo := paths["/echo"].(map[string]any)["post"].(map[string]any)
			responses := echo["responses"].(map[string]any)
			expected := ""
			switch profile {
			case "false-body":
				echo["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"] = false
				expected = "/paths/~1echo/post/requestBody/content/application~1json/schema"
			case "false-response", "false-excludedAPI":
				responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"] = false
				expected = "/paths/~1echo/post/responses/200/content/application~1json/schema"
			case "false-unused-media":
				responses["200"].(map[string]any)["content"].(map[string]any)["application/vnd.example+json"] = map[string]any{"schema": false}
				expected = "/paths/~1echo/post/responses/200/content/application~1vnd.example+json/schema"
			case "false-unused-status", "bad-unused-status":
				schema := any(false)
				if profile == "bad-unused-status" {
					schema = map[string]any{"type": "string", "unknown": true}
				}
				responses["201"] = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": schema}}}
				expected = "/paths/~1echo/post/responses/201/content/application~1json/schema"
				if profile == "bad-unused-status" {
					expected += "/unknown"
				}
			case "3.0":
				doc["openapi"] = "3.0.3"
			case "sse-envelope":
				paths["/watch"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["text/event-stream"].(map[string]any)["schema"] = true
			case "parameter":
				echo["parameters"] = []any{map[string]any{"name": "value", "in": "query", "schema": true}}
			case "unionbranch":
				echo["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"] = map[string]any{"oneOf": []any{true, map[string]any{"type": "string"}}}
			}
			input, _ := json.Marshal(doc)
			r := request(input, "Dto")
			r.Options.Validation = checked
			if profile == "false-excludedAPI" {
				r.Options.ExcludeAPIs = []string{"DefaultApi"}
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(profile, checked, got, err)
			}
			if expected != "" {
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) != 1 {
					t.Fatal(err)
				}
				d := ds[0]
				if d.Pointer == nil || *d.Pointer != expected {
					t.Fatal(profile, d)
				}
			}
		}
	}
}
func TestBooleanHTTPReferenceSourceHeight(t *testing.T) {
	for _, edges := range []int{48, 49} {
		responses := map[string]any{"Z": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": true}}}}
		target := "Z"
		for i := edges - 2; i >= 0; i-- {
			name := fmt.Sprintf("R%02d", i)
			responses[name] = map[string]any{"$ref": "#/components/responses/" + target}
			target = name
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"responses": responses}, "paths": map[string]any{"/x": map[string]any{"get": map[string]any{"responses": map[string]any{"200": map[string]any{"$ref": "#/components/responses/" + target}}}}}})
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if edges == 48 && err != nil || edges == 49 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(edges, got, err)
		}
	}
}
