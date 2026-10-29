package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleUnionSources(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 8, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		union := map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}, "nullable": nullable}
		var member any = union
		if mode&1 != 0 {
			member = map[string]any{"$ref": "#/components/schemas/Choice"}
		}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": []any{"ready", 1}}
		if mode&2 != 0 {
			tuple["items"], tuple["default"] = true, []any{map[string]any{"oneOf": "data"}, nil}
		}
		if mode&4 != 0 {
			tuple["items"], tuple["default"] = map[string]any{"type": "integer"}, []any{7}
		}
		invalid := false
		if mode&8 != 0 {
			union["unknownRole"] = true
			invalid = true
		}
		if mode&16 != 0 {
			union["oneOf"] = []any{}
			invalid = true
		}
		if mode&32 != 0 && mode&6 == 0 {
			tuple["default"] = []any{false}
			invalid = true
		}
		if mode&64 != 0 {
			tuple["prefixItems"] = []any{member, member}
			if mode&6 == 0 {
				tuple["default"] = []any{map[string]any{"data": 1}}
				invalid = mode&24 != 0 || mode&1 != 0
			}
		}
		if mode&128 != 0 {
			tuple["nullable"] = true
		}
		catalog := map[string]any{"Rows": tuple}
		if mode&1 != 0 {
			catalog["Choice"] = union
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
			t.Fatal("union tuple nondeterminism", mode)
		}
		if (failure != nil) != invalid || failure != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(mode, nullable, checked, failure)
		}
	})
}
