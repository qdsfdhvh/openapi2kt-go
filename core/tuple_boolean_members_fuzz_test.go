package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleBooleanMembers(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(3), true, true)
	f.Add(uint8(17), false, true)
	f.Fuzz(func(t *testing.T, mode uint8, nullable, checked bool) {
		prefix := []any{false}
		var literal any = []any{}
		if mode&1 != 0 {
			prefix = []any{true}
			literal = []any{nil, map[string]any{"default": "data"}}
		}
		if mode&2 != 0 {
			prefix = append(prefix, true)
			literal = []any{nil, true}
		}
		tuple := map[string]any{"type": "array", "prefixItems": prefix, "default": literal}
		if nullable {
			tuple["nullable"] = true
		}
		switch (mode >> 2) % 3 {
		case 1:
			tuple["items"] = false
		case 2:
			tuple["items"] = true
			tuple["default"] = []any{nil, map[string]any{"enum": "data"}}
		}
		if mode&16 != 0 {
			tuple["default"] = true
		}
		input, err := json.Marshal(tuple)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(schemas(`{"Rows":`+string(input)+`}`), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("Boolean tuple nondeterminism")
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial Boolean tuple output")
		}
		if mode&16 == 0 && err != nil || mode&16 != 0 && err == nil {
			t.Fatal("Boolean tuple literal role", mode, err)
		}
		rq.Document = schemas(`{"Holder":{"properties":{"rows":` + string(input) + `},"required":["rows"]}}`)
		_, required := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (required == nil) {
			t.Fatal("required suppression changed Boolean tuple annotation validation", err, required)
		}
	})
}
