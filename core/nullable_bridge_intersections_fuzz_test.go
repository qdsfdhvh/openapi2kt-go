package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNullableNamedDTOBridgeIntersection(f *testing.F) {
	f.Add(uint8(0), false, false, false)
	f.Add(uint8(7), true, true, true)
	f.Fuzz(func(t *testing.T, count uint8, checked, targetNullable, memberFalse bool) {
		fields := map[string]any{}
		target := "Value"
		for i := 0; i < int(count%12)+1; i++ {
			fields[fmt.Sprint("h", i)] = map[string]any{"$ref": "#/components/schemas/" + target}
			target = fmt.Sprint("Hops/properties/h", i)
		}
		value := map[string]any{"nullable": targetNullable, "properties": map[string]any{"id": map[string]any{"type": "string"}}}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Value": value, "Hops": map[string]any{"properties": fields}, "Bridge": map[string]any{"$ref": "#/components/schemas/" + target}, "Consumer": map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/Bridge", "nullable": !memberFalse}, map[string]any{"nullable": true, "properties": map[string]any{"extra": map[string]any{"type": "boolean"}}}}}}}}
		data, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = checked
		first, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal("finite original named bridge rejected", e)
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("bridge nondeterminism", e)
		}
	})
}
