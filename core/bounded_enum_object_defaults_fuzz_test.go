package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzBoundedEnumObjectDefaults(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("é😀", uint8(3))
	f.Add("one", uint8(16))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 {
			t.Skip()
		}
		other := word + "other"
		leaf := map[string]any{"type": "string", "format": "date", "enum": []any{word, other}, "minLength": int(mode % 4)}
		root := map[string]any{"properties": map[string]any{"state": leaf}, "required": []string{"state"}, "default": map[string]any{"state": word}}
		consumer := word
		invalid := utf8.RuneCountInString(word) < int(mode%4)
		if mode&16 != 0 {
			consumer, invalid = "future-"+word, true
		}
		if mode&32 != 0 {
			root["nullable"], root["default"] = true, nil
		}
		data, err := json.Marshal(map[string]any{
			"Value":  root,
			"Alias":  map[string]any{"$ref": "#/components/schemas/Value", "default": map[string]any{"state": other}},
			"Holder": map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Alias", "default": map[string]any{"state": consumer}}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(schemas(string(data)), "Dto")
		rq.Options.Validation = true
		if mode&64 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) {
			t.Fatal("bounded object factory/default ownership nondeterminism")
		}
		if (failure != nil) != invalid {
			t.Fatal("bounded object factory/default contract", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.HasSuffix(*ds[0].Pointer, "/default/state") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("bounded object annotation lost strict original value Pointer", failure)
			}
		}
	})
}
