package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzFalseDTOOwnerAliasDefaults(f *testing.F) {
	for _, seed := range [][3]uint16{{1, 0, 0}, {239, 2, 1}, {240, 4, 0}, {10, 1, 0}, {10, 3, 1}, {10, 7, 2}} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, width, mode, style uint16) {
		count := int(width%243) + 1
		target := map[string]any{"properties": map[string]any{"rows": map[string]any{"type": "array", "items": false}}}
		if style&2 != 0 {
			target["maxProperties"] = 1
		}
		if mode&4 != 0 {
			target["default"] = map[string]any{"rows": []any{}}
		}
		value := map[string]any{"rows": []any{}}
		if mode%4 == 1 {
			value["rows"] = []any{nil}
		}
		props, values := map[string]any{}, map[string]any{}
		names := []string{}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("p%03d", i)
			props[name] = map[string]any{"$ref": "#/components/schemas/Alias"}
			values[name] = value
			names = append(names, name)
		}
		owner := map[string]any{"properties": props, "default": values}
		if mode%4 == 2 {
			owner["required"] = names
		} else if mode%4 == 3 {
			owner["nullable"] = true
			owner["default"] = nil
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Alias": map[string]any{"$ref": "#/components/schemas/Target"}, "Outer": owner}}})
		before := append([]byte(nil), input...)
		rq := request(input, "Dto")
		rq.Options.Validation = style&2 != 0
		if style&1 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, err := core.GenerateWithDiagnostics(rq)
		again, e := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, before) || !reflect.DeepEqual(report, again) || !reflect.DeepEqual(err, e) {
			t.Fatal("root-member planning changed input or was nondeterministic", err, e)
		}
		positive := count <= 240 && mode%4 != 1
		if positive != (err == nil) {
			t.Fatal("root-member acceptance mismatch", count, mode, style, err)
		}
		if err != nil && !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("partial root-member artifacts", err)
		}
	})
}
