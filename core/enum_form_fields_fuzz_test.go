package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzEnumFormFields(f *testing.F) {
	f.Add("field", uint8(0), false, false)
	f.Add("class", uint8(3), true, true)
	f.Fuzz(func(t *testing.T, wire string, kind uint8, bodyRequired, checked bool) {
		if len(wire) > 96 || wire == "" {
			t.Skip()
		}
		field := map[string]any{"type": "string", "enum": []string{"ready", "done"}}
		switch kind % 6 {
		case 1:
			field = map[string]any{"type": "boolean", "enum": []bool{true}}
		case 2:
			field = map[string]any{"type": "integer", "enum": []int{1, 2}}
		case 3:
			field = map[string]any{"type": "integer", "format": "int64", "enum": []int{1, 2}}
		case 4:
			field = map[string]any{"type": "number", "format": "float", "enum": []float64{0, 1.25}}
		case 5:
			field = map[string]any{"type": "number", "enum": []float64{0, 2.5}}
		}
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
			t.Fatal("enum form nondeterminism", e)
		}
		// A matching required field must have the same selected-source admission.
		schema["required"] = []string{wire}
		b, _ = json.Marshal(schema)
		rq.Document = formRequestDocument(string(b), bodyRequired)
		if _, e = core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("enum form changed builtin source admission", e)
		}
	})
}
