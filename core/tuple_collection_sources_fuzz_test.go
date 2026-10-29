package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleCollectionSources(f *testing.F) {
	for _, mode := range []uint8{0, 1, 2, 5, 16, 32, 64, 128, 255} {
		f.Add(mode, false, true)
	}
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		leaf := map[string]any{"type": "integer"}
		member := map[string]any{"type": "array", "items": leaf}
		var inner any = []any{1}
		if mode&1 != 0 {
			member = map[string]any{"additionalProperties": leaf}
			inner = map[string]any{"default": 1}
		}
		original := member
		if mode&2 != 0 {
			member = map[string]any{"$ref": "#/components/schemas/Values"}
		}
		prefix := []any{member}
		raw := false
		if mode&4 != 0 {
			prefix = append(prefix, map[string]any{"type": "string"})
			raw = true
		}
		tuple := map[string]any{"type": "array", "prefixItems": prefix}
		if mode&8 != 0 {
			tuple["items"], raw = true, true
		}
		var value any = []any{inner}
		if raw {
			value = []any{nil, map[string]any{"items": "data"}}
		}
		invalid := false
		if mode&16 != 0 && !raw {
			value = []any{true}
			invalid = true
		}
		if nullable {
			tuple["nullable"] = true
			if mode&32 != 0 {
				value, invalid = nil, false
			}
		}
		tuple["default"] = value
		if mode&64 != 0 {
			original["unknownTupleSource"], invalid = true, true
		}
		if mode&128 != 0 {
			leaf["enum"], leaf["default"], invalid = []any{1, 2}, "bad", true
		}
		catalog := map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}, "Rows": tuple}
		if mode&2 != 0 {
			catalog["Values"] = original
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("collection tuple nondeterminism", mode)
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial collection tuple output")
		}
		if (err != nil) != invalid {
			t.Fatal("collection source/default/suppression", mode, nullable, checked, err)
		}
	})
}
