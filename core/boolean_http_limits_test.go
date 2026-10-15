package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func TestBooleanHTTPMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		max := 239
		if stream {
			max = 238
		}
		for _, count := range []int{max, max + 1} {
			params := []any{}
			for i := 0; i < count; i++ {
				params = append(params, map[string]any{"name": fmt.Sprintf("q%03d", i), "in": "query", "required": true, "schema": map[string]any{"type": "string"}})
			}
			media := "application/json"
			schema := any(true)
			if stream {
				media = "text/event-stream"
				schema = map[string]any{"oneOf": []any{map[string]any{"type": "object", "required": []any{"event", "data"}, "properties": map[string]any{"event": map[string]any{"type": "string", "const": "sample"}, "data": true}}}}
			}
			op := map[string]any{"parameters": params, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": true}}}, "responses": map[string]any{"200": map[string]any{"content": map[string]any{media: map[string]any{"schema": schema}}}}}
			input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/x": map[string]any{"post": op}}})
			for _, checked := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if count == max && err != nil || count > max && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(stream, count, checked, got, err)
				}
			}
		}
	}
}
