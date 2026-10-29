package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleHierarchyReferences(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		catalog := map[string]any{}
		if err := json.Unmarshal([]byte(`{`+tupleHierarchyCatalog+`}`), &catalog); err != nil {
			t.Fatal(err)
		}
		target := "Root"
		var value any = map[string]any{"kind": "group", "sub": "first", "id": "ready"}
		if mode&1 != 0 {
			target = "Nullable"
		}
		if mode&2 != 0 {
			target = "First"
			value = map[string]any{"id": "ready"}
		}
		member := map[string]any{"$ref": "#/components/schemas/" + target, "nullable": nullable}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": []any{value}}
		if mode&4 != 0 {
			tuple["items"], tuple["default"] = true, []any{map[string]any{"$ref": "data"}, nil}
		}
		if mode&8 != 0 {
			tuple["items"], tuple["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		invalid := false
		if mode&16 != 0 {
			member["default"] = nil
			invalid = true
		}
		if mode&32 != 0 {
			catalog["Root"].(map[string]any)["unknownRole"] = true
			invalid = true
		}
		if mode&64 != 0 && mode&12 == 0 {
			tuple["default"] = []any{map[string]any{"kind": "future"}}
			invalid = true
		}
		if mode&128 != 0 {
			tuple["prefixItems"] = []any{member, map[string]any{"type": "integer"}}
			if mode&12 == 0 {
				tuple["default"] = []any{map[string]any{"data": 1}, nil}
				invalid = mode&48 != 0
			}
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
			t.Fatal("hierarchy tuple nondeterminism", mode)
		}
		if (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(mode, nullable, checked, failure)
		}
	})
}
