package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzRawTupleDefaults(f *testing.F) {
	f.Add(`[null,true,{"default":{"$ref":"data"}},[1.0,"1"]]`, uint8(0), false, false)
	f.Add(`[0,-0e0]`, uint8(1), false, true)
	f.Add(`null`, uint8(2), true, true)
	f.Fuzz(func(t *testing.T, literal string, tail uint8, nullable, checked bool) {
		if len(literal) > 1024 {
			t.Skip()
		}
		var value any
		decoder := json.NewDecoder(bytes.NewBufferString(literal))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			t.Skip()
		}
		schema := map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}, "default": value}
		if nullable {
			schema["nullable"] = true
		}
		switch tail % 3 {
		case 1:
			schema["items"] = true
			if checked {
				schema["uniqueItems"] = true
			}
		case 2:
			schema["items"] = false
		}
		encoded, err := json.Marshal(schema)
		if err != nil {
			t.Skip()
		}
		rq := request(schemas(`{"Rows":`+string(encoded)+`}`), "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		again, repeat := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(err, repeat) {
			t.Fatal("raw tuple default nondeterminism")
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial raw tuple artifacts")
		}
		rq.Document = schemas(`{"Holder":{"properties":{"rows":` + string(encoded) + `},"required":["rows"]}}`)
		_, required := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (required == nil) {
			t.Fatal("required suppression changed original raw literal validation", err, required)
		}
	})
}
