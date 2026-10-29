package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzBoundedEnumCollectionRoots(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("é😀", uint8(17))
	f.Add("one", uint8(64))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 {
			t.Skip()
		}
		minimum := int(mode % 4)
		leaf := map[string]any{"type": "string", "enum": []any{word, word + "other"}, "minLength": minimum}
		if mode&16 != 0 {
			leaf["format"] = "date"
		}
		var shape any = leaf
		var rootValue, aliasValue any = word + "other", word
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&32 != 0 {
			aliasValue, invalid = "future-"+word, true
		}
		for edge := 0; edge < 1+int(mode%4); edge++ {
			if (int(mode)>>edge)&1 == 0 {
				shape = map[string]any{"type": "array", "items": shape}
				rootValue, aliasValue = []any{rootValue}, []any{aliasValue}
			} else {
				shape = map[string]any{"additionalProperties": shape}
				rootValue, aliasValue = map[string]any{"a/~": rootValue}, map[string]any{"a/~": aliasValue}
			}
		}
		tree := shape.(map[string]any)
		tree["default"] = rootValue
		if mode&64 != 0 {
			tree["nullable"] = true
		}
		data, err := json.Marshal(map[string]any{
			"Tree":  tree,
			"Alias": map[string]any{"$ref": "#/components/schemas/Tree", "nullable": false, "default": aliasValue},
			"Use":   map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Alias", "default": rootValue}}, "default": map[string]any{"value": rootValue}},
		})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(schemas(string(data)), "Dto")
		rq.Options.Validation = true
		if mode&128 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) {
			t.Fatal("bounded enum root/default ownership nondeterminism")
		}
		if (failure != nil) != invalid {
			t.Fatal("bounded root alias strict original default", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Alias/default/") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("bounded alias default lost consuming Pointer", failure)
			}
		}
	})
}
