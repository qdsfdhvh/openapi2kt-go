package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzNestedStringEnumBounds(f *testing.F) {
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
		var value any = word
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&32 != 0 {
			value, invalid = "future-"+word, true
		}
		for edge := 0; edge < 2+int(mode%4); edge++ {
			if (int(mode)>>edge)&1 == 0 {
				shape = map[string]any{"type": "array", "items": shape}
				value = []any{value}
			} else {
				shape = map[string]any{"additionalProperties": shape}
				value = map[string]any{"a/~": value}
			}
		}
		field := shape.(map[string]any)
		if mode&64 != 0 {
			field["nullable"], value, invalid = true, nil, false
		}
		field["default"] = value
		data, err := json.Marshal(field)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(fieldDoc(string(data)), "Dto")
		rq.Options.Validation = true
		if mode&128 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		got, failure := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(failure, repeated) {
			t.Fatal("nested enum constraints/default planning is nondeterministic")
		}
		if (failure != nil) != invalid {
			t.Fatal("nested strict membership/codepoint/default contract", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.Contains(*ds[0].Pointer, "/default/") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("nested enum default lost original value Pointer", failure)
			}
		}
	})
}
