package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzDTONativeCollectionReferenceUnique(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, nullable, optimized bool) {
		leafValue := map[string]any{"type": "integer", "enum": []any{0, 1}}
		var value any = 0
		switch variant % 5 {
		case 1:
			leafValue["format"] = "int64"
		case 2:
			leafValue["type"] = "number"
			leafValue["format"] = "float"
		case 3:
			leafValue["type"] = "number"
			leafValue["format"] = "double"
		case 4:
			leafValue["type"] = "boolean"
			leafValue["enum"] = []any{true, false}
			value = true
		}
		leaf := map[string]any{"type": "array", "items": leafValue, "nullable": nullable}
		var annotation any = []any{value}
		if variant%2 == 1 {
			leaf = map[string]any{"type": "object", "additionalProperties": leafValue, "nullable": nullable}
			annotation = map[string]any{"x": value}
		}
		definitions := map[string]any{"Value": leaf}
		last := "Value"
		for i := range int(variant % 5) {
			name := fmt.Sprint("Alias", i)
			definitions[name] = map[string]any{"$ref": "#/components/schemas/" + last}
			last = name
		}
		field := map[string]any{"$ref": "#/components/schemas/" + last, "default": annotation}
		values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true, "default": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}}}
		definitions["Item"] = map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "value": field}, "required": []string{"id"}}
		definitions["Holder"] = map[string]any{"properties": map[string]any{"values": values}}
		encode := func() []byte {
			b, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": definitions}})
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
			t.Fatal("native enum collection reference getter nondeterminism", e)
		}
		values["default"] = []any{map[string]any{"id": "a"}, map[string]any{"id": "a", "value": annotation}}
		rq.Document = encode()
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("native enum collection reference initializer key mismatch", e)
		}
	})
}
