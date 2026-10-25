package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzNullComponentFactories(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(7), true, true)
	f.Fuzz(func(t *testing.T, variant uint8, enum, optimized bool) {
		leaf := map[string]any{"type": "null", "default": nil}
		if enum {
			leaf = map[string]any{"enum": []any{nil}, "default": nil}
		}
		definitions := map[string]any{"Value": leaf}
		last := "Value"
		for i := range int(variant % 5) {
			name := "Alias" + string(rune('A'+i))
			definitions[name] = map[string]any{"$ref": "#/components/schemas/" + last, "default": nil}
			last = name
		}
		encode := func() []byte {
			b, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": definitions}})
			if e != nil {
				t.Fatal(e)
			}
			return b
		}
		rq := request(encode(), "Dto")
		rq.Options.Validation = variant%2 == 1
		if variant%3 == 1 {
			rq.Options.ModelStyle = core.PlainClass
		}
		rq.Options.DeduplicateModels = optimized
		rq.Options.HoistSharedShapes = optimized
		rq.Options.MergeIdenticalModels = optimized
		rq.Options.LockHoistedNames = optimized
		a, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		b, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(a, b) {
			t.Fatal("nullfactory nondeterminism", e)
		}
		definitions[last].(map[string]any)["default"] = "null"
		rq.Document = encode()
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+last+"/default" || len(r.Artifacts.Files) != 0 {
			t.Fatal("nullfactory coerced literal or lost Pointer", e)
		}
	})
}
