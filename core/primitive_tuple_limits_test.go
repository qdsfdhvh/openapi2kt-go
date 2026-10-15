package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveTupleCheckedEdgesAndAliasConsumerHeight(t *testing.T) {
	for _, edges := range []int{16, 17} {
		var leaf any = map[string]any{"type": "array", "minItems": 1, "prefixItems": []any{map[string]any{"type": "string"}}}
		for i := 1; i < edges; i++ {
			leaf = map[string]any{"type": "array", "minItems": 1, "items": leaf}
		}
		r := tupleRequest(t, map[string]any{"properties": map[string]any{"v": leaf}, "required": []string{"v"}})
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(edges, err)
		}
	}
	for _, height := range []int{46, 47} {
		var alias any = map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}, map[string]any{"type": "boolean"}}}
		for i := 1; i < height; i++ {
			alias = map[string]any{"type": "array", "items": alias}
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"ZAlias": alias, "AHolder": map[string]any{"properties": map[string]any{"v": map[string]any{"$ref": "#/components/schemas/ZAlias"}}, "required": []string{"v"}}}}})
		if err != nil {
			t.Fatal(err)
		}
		got, err := core.GenerateWithDiagnostics(request(input, ""))
		if height == 46 && err != nil || height == 47 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(height, err)
		}
	}
}
