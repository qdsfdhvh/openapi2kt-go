package core_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzCheckedStringEnumRoots(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("é😀", uint8(17))
	f.Add("", uint8(255))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 || !utf8.ValidString(word) {
			t.Skip()
		}
		minimum := int(mode % 4)
		leaf := map[string]any{"type": "string", "enum": []string{word, word + "other"}, "minLength": minimum, "default": word + "other"}
		value := word
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&16 != 0 {
			leaf["format"] = "date"
		}
		if mode&32 != 0 {
			value, invalid = "future-"+word, true
		}
		alias := map[string]any{"$ref": "#/components/schemas/Word", "default": value}
		if mode&64 != 0 {
			leaf["nullable"], alias["nullable"] = true, false
		}
		data, err := json.Marshal(map[string]any{"Word": leaf, "Alias": alias, "Use": map[string]any{"properties": map[string]any{"word": map[string]any{"$ref": "#/components/schemas/Alias", "default": word + "other"}}}})
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
			t.Fatal("checked root lost strict independent deterministic defaults", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Alias/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("checked root default lost original alias Pointer", failure)
			}
		}
	})
}
