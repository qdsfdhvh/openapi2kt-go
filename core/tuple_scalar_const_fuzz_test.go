package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleScalarConst(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 3, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		var pin any = "fixed"
		member := map[string]any{"const": pin}
		switch mode & 3 {
		case 1:
			pin = 7
			member = map[string]any{"const": pin}
		case 2:
			pin = 1.5
			member = map[string]any{"type": "number", "format": "float", "const": pin}
		case 3:
			pin = "7"
			member = map[string]any{"x-native-type": "bigint", "const": pin}
		}
		original := member
		member["nullable"] = nullable
		if mode&4 != 0 {
			delete(original, "nullable")
			member = map[string]any{"$ref": "#/components/schemas/Only", "nullable": nullable}
		}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": []any{pin}}
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
			member["default"], invalid = pin, true
		}
		if mode&128 != 0 && mode&24 == 0 {
			tuple["default"], invalid = []any{false}, true
		}
		catalog := map[string]any{"Rows": tuple, "Holder": map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}}
		if mode&4 != 0 {
			catalog["Only"] = original
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
			t.Fatal("const source nondeterminism", mode)
		}
		if (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("const source/default/erasure", mode, nullable, checked, failure)
		}
	})
}
