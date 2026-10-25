package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzDTOJsonNullUnique(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(4), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, nullable, optimized bool) {
		field := map[string]any{}
		switch variant % 5 {
		case 0:
			field["type"] = "null"
			field["default"] = nil
		case 1:
			field["enum"] = []any{nil}
			field["default"] = nil
		case 2:
			field["const"] = nil
		case 3:
			field["type"] = "null"
			field["const"] = nil
		case 4:
			field["type"] = "null"
			field["enum"] = []any{nil}
			field["default"] = nil
		}
		if nullable {
			field["nullable"] = true
		}
		catalog := map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "pin": field}, "required": []string{"id"}}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"$ref": "#/components/schemas/Item"}}}}}
		encode := func() []byte {
			b, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
			if e != nil {
				t.Fatal(e)
			}
			return b
		}
		rq := request(encode(), "Dto")
		rq.Options.Validation = true
		if variant%2 == 1 {
			rq.Options.ModelStyle = core.PlainClass
		}
		rq.Options.DeduplicateModels = optimized
		rq.Options.HoistSharedShapes = optimized
		rq.Options.MergeIdenticalModels = optimized
		rq.Options.LockHoistedNames = optimized
		first, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("nullgetter nondeterminism", e)
		}
		pointer := "/components/schemas/Item/properties/pin/default"
		if _, ok := field["enum"]; ok {
			field["enum"] = []any{nil, nil}
			pointer = "/components/schemas/Item/properties/pin/enum/1"
		} else {
			field["default"] = "null"
		}
		rq.Document = encode()
		bad, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(bad.Artifacts.Files) != 0 {
			t.Fatal("nullsource failure lost originalPointer", e)
		}
	})
}
