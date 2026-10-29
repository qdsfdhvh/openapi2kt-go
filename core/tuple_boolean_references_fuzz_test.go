package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleBooleanReferences(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 12, 16, 20, 32, 64, 76, 128, 149} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		ref := func(name string) map[string]any { return map[string]any{"$ref": "#/components/schemas/" + name} }
		first := ref("Empty")
		if mode&1 != 0 {
			first = ref("Any")
		}
		prefix := []any{first}
		if mode&2 != 0 {
			prefix = append(prefix, ref("Other"))
		}
		raw := mode&3 != 0
		var literal any = []any{}
		if raw {
			literal = []any{nil, map[string]any{"default": "data"}}
		}
		tuple := map[string]any{"type": "array", "prefixItems": prefix}
		switch (mode >> 2) % 6 {
		case 1:
			tuple["items"], raw, literal = true, true, []any{nil, true}
		case 2:
			tuple["items"] = false
		case 3:
			tuple["items"], raw, literal = ref("Maybe"), false, []any{nil}
		case 4:
			tuple["items"], raw, literal = ref("Any"), true, []any{nil, 1}
		case 5:
			tuple["items"], raw, literal = ref("Empty"), false, []any{}
		}
		if mode&64 != 0 {
			literal = []any{true}
		}
		invalidValue := mode&64 != 0 && !raw
		if nullable {
			tuple["nullable"] = true
			if mode&32 != 0 {
				literal, invalidValue = nil, false
			}
		}
		tuple["default"] = literal
		if mode&128 != 0 {
			first["unknownTupleSource"] = true
		}
		owner := map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Empty": false, "Other": false, "Any": true, "Maybe": map[string]any{"$ref": "#/components/schemas/Empty", "nullable": true}, "Rows": tuple, "Holder": owner}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("Boolean reference tuple nondeterminism", mode)
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial output after Boolean reference tuple failure")
		}
		if (err != nil) != (invalidValue || mode&128 != 0) {
			t.Fatal("Boolean reference source/default/suppression", mode, nullable, checked, err)
		}
	})
}
