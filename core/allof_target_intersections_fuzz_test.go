package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzAllOfTargetIntersections(f *testing.F) {
	f.Add(uint8(0), false, false, false)
	f.Add(uint8(7), true, true, true)
	f.Fuzz(func(t *testing.T, count uint8, checked, nullable, direct bool) {
		fields := map[string]any{}
		target := "Owner/properties/node"
		for i := 0; i < int(count%12)+1; i++ {
			fields[fmt.Sprint("h", i)] = map[string]any{"$ref": "#/components/schemas/" + target}
			target = fmt.Sprint("Hops/properties/h", i)
		}
		source := "Bridge"
		if direct {
			source = target
		}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{
			"Owner": map[string]any{"properties": map[string]any{"node": map[string]any{"nullable": nullable, "properties": map[string]any{"id": map[string]any{"type": "string"}}}}},
			"Hops":  map[string]any{"properties": fields}, "Bridge": map[string]any{"$ref": "#/components/schemas/" + target},
			"Consumer": map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/" + source}, map[string]any{"nullable": true, "properties": map[string]any{"extra": map[string]any{"type": "boolean"}}}}}}}}
		catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
		owner := catalog["Owner"].(map[string]any)
		props := owner["properties"].(map[string]any)
		props["node"] = map[string]any{"allOf": []any{props["node"]}}
		data, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = checked
		first, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal("bounded original internal DTO intersection rejected", e)
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("internal intersection nondeterminism", e)
		}
	})
}
