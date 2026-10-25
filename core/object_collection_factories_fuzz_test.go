package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzObjectCollectionComponentFactoryValues(f *testing.F) {
	f.Add(uint8(0), false, false, false)
	f.Add(uint8(2), true, true, true)
	f.Fuzz(func(t *testing.T, kind uint8, bad, nullable, checked bool) {
		leaf := map[string]any{"properties": map[string]any{"value": map[string]any{"type": "string"}}, "required": []string{"value"}}
		var value any = "ab"
		if bad {
			value = 1
		}
		row := map[string]any{"value": value}
		var shape map[string]any
		var literal any
		pointer := "/components/schemas/Rows/default"
		switch kind % 3 {
		case 0:
			shape = map[string]any{"type": "array", "items": leaf}
			literal = []any{row}
			pointer += "/0/value"
		case 1:
			shape = map[string]any{"type": "object", "additionalProperties": leaf}
			literal = map[string]any{"a/~": row}
			pointer += "/a~1~0/value"
		case 2:
			shape = map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "array", "items": leaf}}}
			literal = []any{map[string]any{"a/~": []any{row}}}
			pointer += "/0/a~1~0/0/value"
		}
		shape["nullable"] = nullable
		if nullable && !bad {
			literal = nil
		}
		shape["default"] = literal
		data, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Rows": shape}}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != pointer {
				t.Fatal("bad factory member lost Pointer", err, pointer)
			}
			return
		}
		if err != nil {
			t.Fatal("ordinary DTO collection factory rejected", err)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("collection factory nondeterminism", err)
		}
	})
}
