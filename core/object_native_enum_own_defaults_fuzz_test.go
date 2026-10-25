package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzObjectNativeEnumOwnDefaultAliasValues(f *testing.F) {
	f.Add(uint8(0), uint8(3), false, true)
	f.Add(uint8(3), uint8(5), true, false)
	f.Fuzz(func(t *testing.T, backing, count uint8, bad, checked bool) {
		kinds := []string{"boolean", "integer", "integer", "number", "number"}
		formats := []string{"", "int32", "int64", "float", "double"}
		vocabularies := [][]any{{false, true}, {-2147483648, 2147483647}, {int64(-9223372036854775808), int64(9223372036854775807)}, {16777216, 16777217}, {1.5, 2.5}}
		i := int(backing % 5)
		property := map[string]any{"type": kinds[i], "enum": vocabularies[i], "default": vocabularies[i][0]}
		if formats[i] != "" {
			property["format"] = formats[i]
		}
		components := map[string]any{"Target": map[string]any{"properties": map[string]any{"value": property}, "default": map[string]any{}}}
		previous, last := "Target", ""
		for j := 0; j <= int(count%6); j++ {
			last = fmt.Sprintf("Alias%d", j)
			var value any = map[string]any{}
			if j%2 == 0 {
				value = nil
			}
			components[last] = map[string]any{"$ref": "#/components/schemas/" + previous, "nullable": j%2 == 0, "default": value}
			previous = last
		}
		if bad {
			components[last].(map[string]any)["default"] = map[string]any{"value": "wrong-native-kind"}
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}})
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(first.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/"+last+"/default/value" {
				t.Fatal("wrong native alias value lost exact Pointer", err)
			}
			return
		}
		if err != nil {
			t.Fatal("native own default alias chain rejected", err)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("native own default alias nondeterminism", err)
		}
	})
}
