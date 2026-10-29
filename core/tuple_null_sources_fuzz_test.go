package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleNullSources(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		member := map[string]any{"type": "null"}
		if mode&1 != 0 {
			member = map[string]any{"enum": []any{nil}}
		}
		if mode&2 != 0 {
			member = map[string]any{"const": nil}
		}
		member["nullable"] = nullable
		original := member
		if mode&4 != 0 {
			member = map[string]any{"$ref": "#/components/schemas/Only", "nullable": nullable}
		}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": []any{nil}}
		if mode&8 != 0 {
			tuple["items"], tuple["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		if mode&16 != 0 {
			tuple["items"], tuple["default"] = true, []any{map[string]any{"const": "data"}, nil}
		}
		invalid := false
		if mode&32 != 0 {
			original["unknownWire"], invalid = true, true
		}
		if mode&64 != 0 {
			original["default"], invalid = nil, true
		}
		if mode&128 != 0 && mode&24 == 0 {
			tuple["default"], invalid = []any{false}, true
		}
		catalog := map[string]any{"Rows": tuple, "Holder": map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}}
		if mode&4 != 0 {
			// Nullable component roots/defaults keep their independent grammar.
			catalog["Only"] = map[string]any{"type": "null"}
			if mode&32 != 0 {
				catalog["Only"].(map[string]any)["unknownWire"] = true
			}
			if mode&64 != 0 {
				member["default"] = nil
			}
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) {
			t.Fatal("JsonNull tuple nondeterminism", mode)
		}
		if (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("null source/default/erasure contract", mode, nullable, checked, failure)
		}
	})
}
