package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzObjectDefaultCollectionGraphs(f *testing.F) {
	f.Add(uint8(3), true, "text", false)
	f.Add(uint8(15), false, "text", true)
	f.Add(uint8(19), true, "", false)
	f.Fuzz(func(t *testing.T, size uint8, checked bool, text string, bad bool) {
		if len(text) > 1024 {
			t.Skip()
		}
		depth := int(size%20) + 1
		schemas := map[string]any{"Leaf": map[string]any{"type": "string"}}
		var value any = text
		if bad {
			value = false
		}
		for i := depth - 1; i >= 0; i-- {
			target := "Leaf"
			if i < depth-1 {
				target = fmt.Sprintf("C%d", i+1)
			}
			node := map[string]any{}
			if i%2 == 0 {
				node["type"] = "array"
				node["items"] = map[string]any{"$ref": "#/components/schemas/" + target}
				value = []any{value}
			} else {
				node["type"] = "object"
				node["additionalProperties"] = map[string]any{"$ref": "#/components/schemas/" + target}
				value = map[string]any{"a/~": value}
			}
			schemas[fmt.Sprintf("C%d", i)] = node
		}
		schemas["Holder"] = map[string]any{"properties": map[string]any{"v": map[string]any{"properties": map[string]any{"member": map[string]any{"$ref": "#/components/schemas/C0"}}, "default": map[string]any{"member": value}}}}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			diagnostics, ok := err.(core.Diagnostics)
			if !ok || len(diagnostics) == 0 || len(first.Artifacts.Files) != 0 {
				t.Fatal("unsafe failure", err)
			}
			return
		}
		if bad {
			t.Fatal("graph accepted Boolean at original String leaf")
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("graph nondeterminism", err)
		}
	})
}
