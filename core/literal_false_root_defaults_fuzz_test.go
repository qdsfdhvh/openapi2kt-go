package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzLiteralFalseRootDefaults(f *testing.F) {
	for _, seed := range [][3]uint16{{1, 0, 0}, {240, 0, 1}, {241, 0, 0}, {10, 1, 0}, {10, 2, 1}, {10, 3, 0}} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, width, mode, style uint16) {
		count := int(width%243) + 1
		props := map[string]any{}
		names := []string{}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("p%03d", i)
			props[name] = false
			names = append(names, name)
		}
		model := map[string]any{"properties": props, "default": map[string]any{}}
		switch mode % 4 {
		case 1:
			model["default"] = map[string]any{names[0]: nil}
		case 2:
			model["required"] = names
		case 3:
			model["nullable"] = true
			model["required"] = names
			model["default"] = nil
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": model}}})
		if err != nil {
			t.Fatal(err)
		}
		before := append([]byte(nil), input...)
		rq := request(input, "Dto")
		if style%2 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, e := core.GenerateWithDiagnostics(rq)
		again, e2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(report, again) || !reflect.DeepEqual(e, e2) {
			t.Fatal("source mutation/nondeterministic factory", e, e2)
		}
		positive := count <= 240 && (mode%4 == 0 || mode%4 == 3)
		if positive != (e == nil) {
			t.Fatal("ownroot factory accept mismatch", count, mode, e)
		}
		if e != nil && !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("partial factory artifacts", e)
		}
	})
}
