package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNullValueDefaults(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, nullable, optimized bool) {
		literal := map[string]any{"enum": []any{nil}}
		if variant%2 == 1 {
			literal["type"] = "null"
		}
		if nullable {
			literal["nullable"] = true
		}
		value := map[string]any{"id": "value"}
		if variant%3 != 0 {
			value["fixed"] = nil
		}
		target := map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "fixed": literal}, "required": []string{"id"}, "default": value}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"value": target}}}}}
		encode := func() []byte {
			b, e := json.Marshal(doc)
			if e != nil {
				t.Fatal(e)
			}
			return b
		}
		rq := request(encode(), "Dto")
		rq.Options.Validation = variant%2 == 0
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
			t.Fatal("nullvalue default nondeterminism", e)
		}
		value["fixed"] = "null"
		rq.Document = encode()
		bad, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/value/default/fixed" || len(bad.Artifacts.Files) != 0 {
			t.Fatal("nullvalue member error Pointer", e)
		}
	})
}
