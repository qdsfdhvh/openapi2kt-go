package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleInlineAllOf(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 4, 12, 16, 33, 64, 129} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		name := map[string]any{"type": "string"}
		first := map[string]any{"properties": map[string]any{"name": name}, "required": []any{"name"}}
		second := map[string]any{"properties": map[string]any{"n": map[string]any{"type": "integer", "default": 7}}}
		if nullable {
			first["nullable"], second["nullable"] = true, true
		}
		item := map[string]any{"allOf": []any{first, second}}
		prefix := []any{item}
		if mode&1 != 0 {
			// JSON encoding produces a distinct original node; shape equality
			// grants no same-source tuple signature.
			prefix = append(prefix, item)
		}
		tuple := map[string]any{"type": "array", "prefixItems": prefix}
		raw := mode&1 != 0
		switch (mode >> 1) % 4 {
		case 1:
			tuple["items"], raw = true, true
		case 2:
			tuple["items"] = false
		case 3:
			tuple["items"], raw = map[string]any{"type": "string"}, true
		}
		var value any = []any{map[string]any{"name": "outer"}}
		if raw {
			if (mode>>1)%4 == 3 {
				value = []any{"tail"}
			} else {
				value = []any{nil, map[string]any{"allOf": "data"}}
			}
		}
		invalid := false
		if mode&16 != 0 && !raw {
			value, invalid = []any{map[string]any{}}, true
		}
		if mode&32 != 0 {
			tuple["nullable"], value, invalid = true, nil, false
		}
		tuple["default"] = value
		if mode&64 != 0 {
			name["unknownTupleMember"], invalid = true, true
		}
		if mode&128 != 0 {
			second["properties"].(map[string]any)["n"].(map[string]any)["default"], invalid = "bad", true
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Rows": tuple, "Holder": map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("inline allOf source/default nondeterminism", mode)
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial artifacts after inline allOf failure")
		}
		if (err != nil) != invalid {
			t.Fatal("inline allOf source/value/suppression", mode, nullable, checked, err)
		}
	})
}
