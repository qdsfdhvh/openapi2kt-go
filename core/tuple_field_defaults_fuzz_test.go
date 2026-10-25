package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleFieldDefaults(f *testing.F) {
	f.Add("value", uint8(0), true)
	f.Add("2026-01-02T03:04:05Z", uint8(2), false)
	f.Fuzz(func(t *testing.T, value string, kind uint8, checked bool) {
		if len(value) > 128 {
			t.Skip()
		}
		leaf := map[string]any{"type": "string"}
		var data any = value
		switch kind % 4 {
		case 1:
			leaf["type"] = "integer"
			data = int(kind)
		case 2:
			leaf["format"] = "date-time"
		case 3:
			leaf["format"] = "uuid"
		}
		field := map[string]any{"type": "array", "prefixItems": []any{leaf, leaf}, "default": []any{data}}
		if checked {
			field["minItems"] = 1
		}
		d := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"values": field}}}}}
		input, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		second, retry := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, retry) || !bytes.Equal(input, original) {
			t.Fatal("tuple field default mutated source or drifted")
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial tuple default error output")
		}
	})
}
