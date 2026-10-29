package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzFormFieldDefaults(f *testing.F) {
	f.Add(uint8(0), `"a b+&雪"`, false, false)
	f.Add(uint8(1), `0`, true, true)
	f.Add(uint8(2), `null`, true, false)
	f.Add(uint8(3), `"unknown"`, false, true)
	f.Fuzz(func(t *testing.T, kind uint8, literal string, required, checked bool) {
		if len(literal) > 512 {
			t.Skip()
		}
		var value any
		decoder := json.NewDecoder(bytes.NewBufferString(literal))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			t.Skip()
		}
		field := map[string]any{"type": "string", "default": value}
		switch kind % 4 {
		case 1:
			field["type"] = "integer"
		case 2:
			field["type"] = "boolean"
		case 3:
			field["enum"] = []string{"ready", "done"}
		}
		schema := map[string]any{"type": "object", "properties": map[string]any{"value": field}}
		if required {
			schema["required"] = []string{"value"}
		}
		input, err := json.Marshal(schema)
		if err != nil {
			t.Skip()
		}
		rq := request(formRequestDocument(string(input), true), "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("form default nondeterminism")
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial default artifacts")
		}
		// Suppressing a required initializer must never suppress annotation
		// validation or alter the scalar/enum source admission.
		if required {
			delete(schema, "required")
		} else {
			schema["required"] = []string{"value"}
		}
		input, _ = json.Marshal(schema)
		rq.Document = formRequestDocument(string(input), true)
		_, opposite := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (opposite == nil) {
			t.Fatal("required suppression changed default validation", err, opposite)
		}
	})
}
