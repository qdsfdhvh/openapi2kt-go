package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzReferencedEnumCollectionFactories(f *testing.F) {
	f.Add("one", "a/~", uint8(0))
	f.Add("é😀", "$ref", uint8(49))
	f.Add("", "default", uint8(255))
	f.Fuzz(func(t *testing.T, word, key string, mode uint8) {
		if len(word) > 96 || len(key) > 64 || !utf8.ValidString(word) || !utf8.ValidString(key) {
			t.Skip()
		}
		var first, second, value any = word, word + "other", word
		leaf := map[string]any{"enum": []any{first, second}, "default": first}
		if mode&2 != 0 {
			first, second, value = 1, 2, 1
			leaf = map[string]any{"type": "integer", "enum": []any{first, second}, "default": first}
		}
		invalid := mode&16 != 0
		if invalid {
			if mode&2 != 0 {
				value = 3
			} else {
				value = word + "future"
			}
		}
		container := func(v any) any { return []any{v} }
		tree := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/WordAlias"}}
		if mode&1 != 0 {
			container = func(v any) any { return map[string]any{key: v} }
			tree = map[string]any{"additionalProperties": map[string]any{"$ref": "#/components/schemas/WordAlias"}}
		}
		tree["default"] = container(first)
		alias := map[string]any{"$ref": "#/components/schemas/Tree", "default": container(value)}
		data, err := json.Marshal(map[string]any{"Word": leaf, "WordAlias": map[string]any{"$ref": "#/components/schemas/Word", "default": second}, "Tree": tree, "Alias": alias})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(schemas(string(data)), "Dto")
		rq.Options.Validation = mode&32 != 0
		if mode&64 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) || (failure != nil) != invalid {
			t.Fatal("reference collection factory lost strict independent deterministic annotations", mode, failure)
		}
		if invalid {
			pointer := "/components/schemas/Alias/default/0"
			if mode&1 != 0 {
				pointer = "/components/schemas/Alias/default/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
			}
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("reference factory lost original default Pointer", pointer, failure)
			}
		}
	})
}
