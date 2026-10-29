package core_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzBorrowedEnumCollectionDefaults(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("é😀", uint8(17))
	f.Add("", uint8(255))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 || !utf8.ValidString(word) {
			t.Skip()
		}
		minimum := int(mode % 4)
		leaf := map[string]any{"enum": []string{word, word + "other"}, "minLength": minimum}
		value := word
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&16 != 0 {
			leaf["format"] = "date"
		}
		if mode&32 != 0 {
			value, invalid = "future-"+word, true
		}
		tree := map[string]any{"type": "array", "items": leaf, "default": []string{word + "other"}}
		alias := map[string]any{"$ref": "#/components/schemas/Owner/properties/tree", "default": []string{value}}
		if mode&64 != 0 {
			tree["nullable"], alias["nullable"] = true, false
		}
		data, err := json.Marshal(map[string]any{"Owner": map[string]any{"properties": map[string]any{"tree": tree}}, "Alias": alias})
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
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) || (failure != nil) != invalid {
			t.Fatal("borrowed collection factory lost strict deterministic annotations", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Alias/default/0" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("borrowed collection default lost original value Pointer", failure)
			}
		}
	})
}
