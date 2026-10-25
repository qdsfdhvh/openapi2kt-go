package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzOptionalFormRequestSourceAdmission(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, kind uint8, bad, checked bool) {
		field := map[string]any{"type": "string"}
		switch kind % 6 {
		case 1:
			field["type"] = "boolean"
		case 2:
			field["type"] = "integer"
			field["format"] = "int32"
		case 3:
			field["type"] = "integer"
			field["format"] = "int64"
		case 4:
			field["type"] = "number"
		case 5:
			field["minLength"] = 1
		}
		required := []string{"field"}
		if bad {
			required = []string{}
		}
		schema := map[string]any{"type": "object", "properties": map[string]any{"field": field}, "required": required}
		encode := func(bodyRequired bool) []byte {
			data, err := json.Marshal(schema)
			if err != nil {
				t.Fatal(err)
			}
			return formRequestDocument(string(data), bodyRequired)
		}
		rq := request(encode(true), "Dto")
		rq.Options.Validation = checked
		_, sourceErr := core.GenerateWithDiagnostics(rq)
		rq.Document = encode(false)
		first, err := core.GenerateWithDiagnostics(rq)
		if sourceErr != nil {
			if len(first.Artifacts.Files) != 0 || !reflect.DeepEqual(err, sourceErr) {
				t.Fatal("optional form changed source admission", err, sourceErr)
			}
			return
		}
		if err != nil {
			t.Fatal("optional valid form rejected", err)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("optional form nondeterminism", err)
		}
	})
}
