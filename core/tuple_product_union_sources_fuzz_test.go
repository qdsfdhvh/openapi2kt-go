package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleProductUnionSources(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		product := map[string]any{}
		if err := json.Unmarshal([]byte(borrowedProductUnionSource()), &product); err != nil {
			t.Fatal(err)
		}
		product["nullable"] = nullable
		var member any = product
		catalog := map[string]any{}
		if mode&1 != 0 {
			catalog["Product"] = product
			catalog["Alias"] = map[string]any{"$ref": "#/components/schemas/Product"}
			member = map[string]any{"$ref": "#/components/schemas/Alias"}
		}
		value := []any{map[string]any{"kind": "id", "id": "ready", "common": "shared"}}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": value}
		if mode&2 != 0 {
			tuple["items"], tuple["default"] = true, []any{map[string]any{"allOf": "data"}, nil}
		}
		if mode&4 != 0 {
			tuple["items"], tuple["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		invalid := false
		if mode&8 != 0 {
			product["unknownRole"] = true
			invalid = true
		}
		if mode&16 != 0 {
			product["allOf"] = []any{}
			invalid = true
		}
		if mode&32 != 0 && mode&6 == 0 {
			tuple["default"] = []any{map[string]any{"kind": "future"}}
			invalid = true
		}
		if mode&64 != 0 && mode&16 == 0 {
			product["allOf"].([]any)[0] = map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}}
			invalid = true
		}
		if mode&128 != 0 {
			tuple["nullable"] = true
		}
		catalog["Rows"] = tuple
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) {
			t.Fatal("product tuple nondeterminism", mode)
		}
		if (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(mode, nullable, checked, failure)
		}
	})
}
