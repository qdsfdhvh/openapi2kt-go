package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleRecursiveCover(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		var root map[string]any
		if err := json.Unmarshal(schemas(`{"Value":`+coverChainRoot("Value", "Value")+tupleCoverAliases+`}`), &root); err != nil {
			t.Fatal(err)
		}
		catalog := root["components"].(map[string]any)["schemas"].(map[string]any)
		target := "Value"
		if mode&1 != 0 {
			target = "Bridge"
		}
		member := map[string]any{"$ref": "#/components/schemas/" + target, "nullable": nullable}
		array := map[string]any{"type": "array", "prefixItems": []any{member}, "default": []any{nil, map[string]any{"$ref": "data", "oneOf": []any{1, true}}, []any{"ready"}}}
		if mode&2 != 0 {
			array["items"] = true
		}
		if mode&4 != 0 {
			array["items"], array["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		if mode&8 != 0 {
			array["prefixItems"] = []any{member, map[string]any{"type": "string"}}
		}
		invalid := mode&48 != 0
		if mode&16 != 0 {
			member["default"] = nil
		}
		if mode&32 != 0 {
			catalog["Value"].(map[string]any)["unknownRole"] = true
		}
		if mode&64 != 0 {
			array["default"] = 7
			invalid = true
		}
		if mode&128 != 0 {
			catalog["Value"].(map[string]any)["oneOf"].([]any)[4].(map[string]any)["items"] = map[string]any{"$ref": "#/components/schemas/Value", "nullable": true}
		}
		catalog["Rows"] = array
		input, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) || (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(mode, nullable, checked, failure, repeated)
		}
	})
}
