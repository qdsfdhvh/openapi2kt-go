package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzDTOStringCollectionUnique(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, nullable, optimized bool) {

		leaf := map[string]any{"type": "string", "enum": []any{"a", "b"}}
		if nullable {
			leaf["nullable"] = true
		}
		var value any = []any{"a", "b"}
		if nullable {
			value = []any{"a", nil, "b"}
		}
		tree := map[string]any{"type": "array", "items": leaf}
		for range int(variant % 4) {
			tree = map[string]any{"type": "array", "items": tree}
			value = []any{value}
		}
		tree["default"] = value
		values := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true, "default": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}}}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "numbers": tree}, "required": []string{"id"}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}}
		encode := func() []byte {
			b, e := json.Marshal(doc)
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
			t.Fatal("collection getter nondeterminism", e)
		}
		values["default"] = []any{map[string]any{"id": "a"}, map[string]any{"id": "a", "numbers": value}}
		rq.Document = encode()
		bad, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(bad.Artifacts.Files) != 0 {
			t.Fatal("constructor collection initializer key mismatch", e)
		}
	})
}
