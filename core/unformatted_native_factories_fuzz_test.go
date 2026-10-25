package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzUnformattedNativeRootFactoryAliases(f *testing.F) {
	f.Add(uint8(0), uint8(0), false, false, false)
	f.Add(uint8(2), uint8(3), true, true, true)
	f.Fuzz(func(t *testing.T, kind, count uint8, bad, nullable, checked bool) {
		nullable = nullable && checked
		types := []string{"integer", "number", "number"}
		vocabs := [][]any{{1, 2}, {1.5, 2.5}, {json.Number("9007199254740993"), json.Number("9007199254740995")}}
		invalid := []any{9, 1.0, json.Number("9007199254740992")}
		i := int(kind % 3)
		var value any = vocabs[i][0]
		if nullable {
			value = nil
		}
		root := map[string]any{"type": types[i], "enum": vocabs[i], "nullable": nullable, "default": value}
		components := map[string]any{"T": root}
		previous := "T"
		for j := 0; j < int(count%4); j++ {
			name := fmt.Sprintf("Alias%d", j)
			components[name] = map[string]any{"$ref": "#/components/schemas/" + previous, "nullable": false, "default": vocabs[i][1]}
			previous = name
		}
		if bad {
			components[previous].(map[string]any)["default"] = invalid[i]
		}
		data, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/"+previous+"/default" {
				t.Fatal("native alias default lost strict vocabulary or Pointer", err)
			}
			return
		}
		if err != nil {
			t.Fatal("unformatted native factory rejected", err)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("native factory nondeterminism", err)
		}
	})
}
