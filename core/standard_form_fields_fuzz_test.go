package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzStandardFormFields(f *testing.F) {
	f.Add("field", uint8(0), false, false)
	f.Add("class", uint8(3), true, true)
	f.Fuzz(func(t *testing.T, wire string, kind uint8, bodyRequired, checked bool) {
		if len(wire) > 96 || wire == "" {
			t.Skip()
		}
		field := map[string]any{"type": "string", "format": []string{"uuid", "date", "date-time"}[kind%3]}
		schema := map[string]any{"type": "object", "properties": map[string]any{wire: field}}
		b, e := json.Marshal(schema)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(formRequestDocument(string(b), bodyRequired), "Dto")
		rq.Options.Validation = checked
		first, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			if len(first.Artifacts.Files) != 0 {
				t.Fatal("partial artifacts", e)
			}
			return
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("standard form nondeterminism", e)
		}
		// A matching required field must have the same selected-source admission.
		schema["required"] = []string{wire}
		b, _ = json.Marshal(schema)
		rq.Document = formRequestDocument(string(b), bodyRequired)
		if _, e = core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("standard form changed builtin source admission", e)
		}
	})
}
