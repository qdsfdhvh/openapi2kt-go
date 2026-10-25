package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzObjectEnumOwnDefaultsAndWrapperSources(f *testing.F) {
	f.Add(uint8(3), "value", false, true)
	f.Add(uint8(7), "雪", true, false)
	f.Fuzz(func(t *testing.T, count uint8, text string, bad, checked bool) {
		if len(text) > 512 || text == "other" {
			t.Skip()
		}
		components := map[string]any{"Target": map[string]any{
			"properties": map[string]any{"mode": map[string]any{"type": "string", "enum": []string{text, "other"}, "default": text}},
			"default":    map[string]any{},
		}}
		previous := "Target"
		last := ""
		for i := 0; i <= int(count%8); i++ {
			last = fmt.Sprintf("Alias%d", i)
			var value any = map[string]any{}
			if i%2 == 0 {
				value = nil
			}
			components[last] = map[string]any{"$ref": "#/components/schemas/" + previous, "nullable": i%2 == 0, "default": value}
			previous = last
		}
		if bad {
			components[last].(map[string]any)["default"] = map[string]any{"mode": text + "-not-in-vocabulary"}
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(first.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/"+last+"/default/mode" {
				t.Fatal("invalid alias annotation lost original value Pointer", err)
			}
			return
		}
		if err != nil {
			t.Fatal("valid original field/default alias chain rejected", err)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("wrapper enum default nondeterminism", err)
		}
	})
}
