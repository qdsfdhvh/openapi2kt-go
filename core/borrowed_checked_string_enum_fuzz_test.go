package core_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzBorrowedCheckedStringEnum(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("é😀", uint8(17))
	f.Add("one", uint8(64))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 {
			t.Skip()
		}
		minimum := int(mode % 4)
		other := word + "other"
		leaf := map[string]any{"type": "string", "enum": []any{word, other}, "minLength": minimum, "default": other}
		if mode&16 != 0 {
			leaf["format"] = "date"
		}
		aliasValue := word
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&32 != 0 {
			aliasValue, invalid = "future-"+word, true
		}
		alias := map[string]any{"$ref": "#/components/schemas/Owner/properties/word", "default": aliasValue}
		if mode&64 != 0 {
			leaf["nullable"], alias["nullable"] = true, false
		}
		data, err := json.Marshal(map[string]any{
			"Owner": map[string]any{"properties": map[string]any{"word": leaf}},
			"Alias": alias,
			"Use":   map[string]any{"properties": map[string]any{"word": map[string]any{"$ref": "#/components/schemas/Alias", "default": other}}, "default": map[string]any{"word": other}},
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
			t.Fatal("checked lexical source/alias/default nondeterminism")
		}
		if (failure != nil) != invalid {
			t.Fatal("checked borrowed enum lost strict independent default", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Alias/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("checked borrowed default lost original alias Pointer", failure)
			}
		}
	})
}
