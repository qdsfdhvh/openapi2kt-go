package core_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"unicode/utf8"

	"openapi-kt/core"
)

func FuzzStringEnumFieldBounds(f *testing.F) {
	f.Add("one", uint8(0))
	f.Add("x", uint8(3))
	f.Add("é😀", uint8(15))
	f.Fuzz(func(t *testing.T, word string, mode uint8) {
		if len(word) > 96 {
			t.Skip()
		}
		minimum := int(mode % 8)
		field := map[string]any{"type": "string", "enum": []any{word}, "minLength": minimum, "default": word}
		switch mode % 3 {
		case 1:
			field["format"] = "date"
		case 2:
			field["x-native-type"] = "org.example.UnusedProvider"
		}
		invalid := utf8.RuneCountInString(word) < minimum
		if mode&8 != 0 {
			field["nullable"], field["default"], invalid = true, nil, false
		}
		if mode&32 != 0 {
			field["default"], invalid = false, true
		}
		holder := map[string]any{"properties": map[string]any{"value": field}}
		if mode&16 != 0 {
			holder["required"] = []string{"value"}
		}
		data, err := json.Marshal(map[string]any{"Holder": holder})
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
			t.Fatal("bounded lexical field nondeterminism")
		}
		if (failure != nil) != invalid {
			t.Fatal("bounded lexical field default contract", mode, failure)
		}
		if invalid {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/value/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("bounded lexical field lost original strict annotation", failure)
			}
		}
	})
}
