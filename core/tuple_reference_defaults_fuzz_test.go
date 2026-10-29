package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleReferenceDefaults(f *testing.F) {
	f.Add("ready", uint8(0), false)
	f.Add("default", uint8(7), true)
	f.Add("other", uint8(16), true)
	f.Fuzz(func(t *testing.T, wire string, mode uint8, checked bool) {
		if len(wire) > 128 {
			t.Skip()
		}
		var target any = map[string]any{"enum": []any{wire}}
		var value any = []any{wire}
		if mode&1 != 0 {
			target = map[string]any{"properties": map[string]any{"name": map[string]any{"type": "string"}}, "required": []any{"name"}}
			value = []any{map[string]any{"name": wire}}
		}
		member := map[string]any{"$ref": "#/components/schemas/Value"}
		tuple := map[string]any{"type": "array", "prefixItems": []any{member}, "default": value}
		switch (mode >> 1) % 3 {
		case 1:
			tuple["items"] = false
		case 2:
			tuple["items"] = member
		}
		if mode&16 != 0 {
			tuple["default"] = true
		}
		document := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Value": target, "Rows": tuple}}}
		input, err := json.Marshal(document)
		if err != nil {
			t.Skip()
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("reference tuple default nondeterminism")
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial reference tuple default failure")
		}
		if mode&16 == 0 && err != nil || mode&16 != 0 && err == nil {
			t.Fatal("reference tuple default literal contract", mode, err)
		}
		document["components"].(map[string]any)["schemas"] = map[string]any{"Value": target, "Holder": map[string]any{"properties": map[string]any{"rows": tuple}, "required": []any{"rows"}}}
		rq.Document, _ = json.Marshal(document)
		_, required := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (required == nil) {
			t.Fatal("required suppression changed original tuple annotation", err, required)
		}
	})
}
