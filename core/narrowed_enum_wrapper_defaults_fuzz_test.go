package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNarrowedEnumWrapperDefaults(f *testing.F) {
	f.Add("on", "off", uint8(0))
	f.Add("null", "future", uint8(15))
	f.Add("same", "same", uint8(4))
	f.Add("a/~", "\"\n", uint8(7))
	f.Fuzz(func(t *testing.T, first, second string, mode uint8) {
		if len(first)+len(second) > 512 {
			t.Skip()
		}
		members, _ := json.Marshal([]any{first, nil, second})
		// The strict loader sees JSON's normalized UTF-8 vocabulary.
		var vocabulary []any
		if err := json.Unmarshal(members, &vocabulary); err != nil {
			t.Fatal(err)
		}
		s := map[string]any{
			"State":  map[string]any{"enum": vocabulary, "default": nil},
			"Middle": map[string]any{"$ref": "#/components/schemas/State", "nullable": true, "default": nil},
			"Narrow": map[string]any{"$ref": "#/components/schemas/Middle", "nullable": false, "default": vocabulary[0]},
			"Again":  map[string]any{"$ref": "#/components/schemas/Narrow", "nullable": true, "default": nil},
			"Bare":   map[string]any{"$ref": "#/components/schemas/Narrow"},
		}
		document := func() []byte {
			data, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": s}})
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
		r := request(document(), "Dto")
		r.Options.Validation = mode&1 != 0
		if mode&2 != 0 {
			r.Options.ModelStyle = core.PlainClass
		}
		r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = mode&4 != 0, mode&4 != 0, mode&4 != 0
		got, err := core.GenerateWithDiagnostics(r)
		if vocabulary[0] == vocabulary[2] {
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("duplicate source admitted a wrapper factory", err)
			}
			return
		}
		if err != nil {
			t.Fatal("valid original vocabulary/nearest alias intent refused", err)
		}
		s["State"].(map[string]any)["nullable"] = true
		r.Document = document()
		control, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("explicit source nullability changed factory report", err)
		}
		for _, value := range []any{nil, map[string]any{"unknown": true}, vocabulary[0].(string) + vocabulary[2].(string) + "!"} {
			if value == vocabulary[0] || value == vocabulary[2] {
				continue
			}
			s["Narrow"].(map[string]any)["default"] = value
			r.Document = document()
			failed, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(failed, core.GenerationReport{}) {
				t.Fatal("strict non-null wrapper lost failure atomicity", value, err)
			}
		}
	})
}
