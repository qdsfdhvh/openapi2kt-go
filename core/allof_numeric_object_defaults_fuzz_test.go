package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzAllOfNumericOuterDefaultConjunction(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(3), true, true)
	f.Fuzz(func(t *testing.T, count uint8, bad, nullable bool) {
		members := []any{}
		for i := 0; i < int(count%4)+1; i++ {
			members = append(members, map[string]any{"properties": map[string]any{"value": map[string]any{"type": "integer", "minimum": -10, "default": -5}}})
		}
		members = append(members, map[string]any{"properties": map[string]any{"value": map[string]any{"type": "integer", "maximum": 10, "default": 5}}})
		var literal any = map[string]any{"value": 0}
		if nullable {
			literal = nil
		}
		if bad {
			literal = map[string]any{"value": -11}
		}
		component := map[string]any{"allOf": members, "default": literal, "nullable": nullable}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Factory": component}}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = true
		first, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(first.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Factory/default/value" {
				t.Fatal("Numeric intersection was dropped", err)
			}
			return
		}
		if err != nil {
			t.Fatal("valid original Numeric contribution group rejected", err)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("Numeric default nondeterminism", err)
		}
	})
}
