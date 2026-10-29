package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleInlineEnums(f *testing.F) {
	f.Add("ready", uint8(0), uint16(1), false)
	f.Add("null", uint8(5), uint16(2), true)
	f.Add("default", uint8(11), uint16(3), true)
	f.Fuzz(func(t *testing.T, wire string, mode uint8, n uint16, checked bool) {
		if len(wire) > 128 {
			t.Skip()
		}
		leaf := map[string]any{"enum": []any{wire}}
		var literal any = []any{wire}
		if mode&1 != 0 {
			leaf = map[string]any{"type": "integer", "enum": []any{int(n), int(n) + 1}}
			literal = []any{int(n)}
		}
		array := map[string]any{"type": "array", "prefixItems": []any{leaf}, "default": literal}
		switch (mode >> 1) % 4 {
		case 1:
			array["items"] = false
		case 2:
			array["items"] = true
			array["default"] = []any{nil, map[string]any{"enum": wire}}
		case 3:
			array["prefixItems"] = []any{leaf, leaf}
			array["default"] = []any{nil, map[string]any{"default": wire}}
		}
		if mode&8 != 0 {
			array["default"] = true
		}
		encoded, err := json.Marshal(array)
		if err != nil {
			t.Skip()
		}
		rq := request(schemas(`{"Rows":`+string(encoded)+`}`), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("inline enum tuple nondeterminism")
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial inline enum tuple failure")
		}
		if mode&8 == 0 && err != nil || mode&8 != 0 && err == nil {
			t.Fatal("inline enum tuple literal contract", mode, err)
		}
		rq.Document = schemas(`{"Holder":{"properties":{"rows":` + string(encoded) + `},"required":["rows"]}}`)
		_, required := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (required == nil) {
			t.Fatal("required suppression changed enum tuple annotation validation", err, required)
		}
	})
}
