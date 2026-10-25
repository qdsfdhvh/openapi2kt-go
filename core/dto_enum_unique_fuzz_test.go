package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzDTOEnumUnique(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, nullable, optimized bool) {
		kind, format := "integer", "int32"
		switch variant % 4 {
		case 1:
			format = "int64"
		case 2:
			kind, format = "number", "float"
		case 3:
			kind, format = "number", "double"
		}
		props := map[string]any{"id": map[string]any{"type": "string"}, "n": map[string]any{"type": kind, "format": format, "default": 0, "enum": []any{0, 1}}}
		props["state"] = map[string]any{"type": "string", "enum": []any{"ready", "done"}, "default": "ready"}
		for i := 0; i < int(variant%5); i++ {
			props[fmt.Sprint("p", i)] = map[string]any{"type": "string"}
		}
		item := map[string]any{"$ref": "#/components/schemas/Item"}
		if nullable {
			item["nullable"] = true
		}
		catalog := map[string]any{"Item": map[string]any{"properties": props, "required": []string{"id"}}, "Holder": map[string]any{"properties": map[string]any{"values": map[string]any{"type": "array", "items": item, "uniqueItems": true, "default": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}}}}}}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = true
		rq.Options.ModelStyle = core.DataClass
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
			t.Fatal("DTOenumuniqueness nondeterminism", e)
		}
		values := catalog["Holder"].(map[string]any)["properties"].(map[string]any)["values"].(map[string]any)
		values["default"] = []any{map[string]any{"id": "a"}, map[string]any{"id": "a", "n": 0}}
		rq.Document, e = json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		bad, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(bad.Artifacts.Files) != 0 {
			t.Fatal("omitted/explicit constructor initializer mismatch", e)
		}
	})
}
