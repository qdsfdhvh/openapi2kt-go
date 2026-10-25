package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzLiteralFalseCollectionDTOFieldDefaults(f *testing.F) {
	for _, seed := range [][3]uint16{{1, 0, 0}, {239, 2, 1}, {240, 4, 0}, {10, 1, 0}, {10, 3, 1}, {10, 7, 2}} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, width, mode, style uint16) {
		count := int(width%243) + 1
		props := map[string]any{}
		names := []string{}
		target := map[string]any{"properties": map[string]any{"rows": map[string]any{"type": "array", "items": false}}}
		var value any = map[string]any{}
		if mode%4 == 1 {
			value = map[string]any{"rows": []any{nil}}
		}
		if mode%4 == 3 {
			target["nullable"] = true
			value = nil
		}
		if mode%8 >= 4 {
			if mode%4 == 3 {
				target["default"] = nil
			} else {
				target["default"] = map[string]any{}
			}
		}
		if style&2 != 0 {
			target["minProperties"] = 0
			target["maxProperties"] = 0
		}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("p%03d", i)
			names = append(names, name)
			props[name] = map[string]any{"$ref": "#/components/schemas/Target", "default": value}
		}
		consumer := map[string]any{"properties": props}
		if mode%4 == 2 {
			consumer["required"] = names
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Consumer": consumer}}})
		if err != nil {
			t.Fatal(err)
		}
		before := append([]byte(nil), input...)
		rq := request(input, "Dto")
		rq.Options.Validation = style&2 != 0
		if style&1 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, e := core.GenerateWithDiagnostics(rq)
		again, e2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(before, input) || !reflect.DeepEqual(report, again) || !reflect.DeepEqual(e, e2) {
			t.Fatal("input changed/nondeterministic", e, e2)
		}
		positive := count <= 240 && mode%4 != 1
		if positive != (e == nil) {
			t.Fatal("field default accept mismatch", count, mode, style, e)
		}
		if e != nil && !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("partial default artifacts", e)
		}
	})
}
