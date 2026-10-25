package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzLiteralFalseCollectionDefaults(f *testing.F) {
	for _, seed := range [][4]uint16{{1, 1, 0, 0}, {16, 15, 0, 3}, {16, 16, 0, 2}, {3, 3, 1, 0}, {3, 3, 2, 0}, {3, 3, 3, 0}, {3, 3, 4, 0}} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}
	f.Fuzz(func(t *testing.T, width, levels, mode, style uint16) {
		count := int(width%16) + 1
		depth := int(levels%18) + 1
		checked := style&2 != 0
		var leaf any = false
		for i := 0; i < depth; i++ {
			layer := map[string]any{"type": "array", "items": leaf}
			if i%2 == 1 {
				layer = map[string]any{"additionalProperties": leaf}
				if checked {
					layer["maxProperties"] = 1
				}
			} else if checked {
				layer["maxItems"] = 1
			}
			leaf = layer
		}
		field := leaf.(map[string]any)
		var value any = []any{}
		if depth%2 == 0 {
			value = map[string]any{}
		}
		switch mode % 4 {
		case 1:
			if depth%2 == 0 {
				value = map[string]any{"bad": nil}
			} else {
				value = []any{nil}
			}
		case 3:
			field["nullable"] = true
			value = nil
		}
		field["default"] = value
		props := map[string]any{}
		names := []string{}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("p%03d", i)
			props[name] = field
			names = append(names, name)
		}
		model := map[string]any{"properties": props}
		if mode%4 == 2 {
			model["required"] = names
		}
		version := "3.1.0"
		if mode&4 != 0 {
			version = "3.0.3"
		}
		input, err := json.Marshal(map[string]any{"openapi": version, "components": map[string]any{"schemas": map[string]any{"Holder": model}}})
		if err != nil {
			t.Fatal(err)
		}
		before := append([]byte(nil), input...)
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		if style&1 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, e := core.GenerateWithDiagnostics(rq)
		again, e2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(report, again) || !reflect.DeepEqual(e, e2) {
			t.Fatal("source mutation/nondeterminism", e, e2)
		}
		positive := mode&4 == 0 && mode%4 != 1 && (!checked || depth <= 16)
		if positive != (e == nil) {
			t.Fatal("ownliteral role/depth accept mismatch", count, depth, mode, style, e)
		}
		if e != nil && !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("partial literal artifacts", e)
		}
	})
}
