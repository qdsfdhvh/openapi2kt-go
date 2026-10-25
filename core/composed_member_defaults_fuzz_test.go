package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzComposedMemberDefaultsPins(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(5), true, true)
	f.Fuzz(func(t *testing.T, backing uint8, bad, checked bool) {
		kinds := []string{"string", "boolean", "integer", "integer", "number", "number"}
		formats := []string{"", "", "int32", "int64", "float", "double"}
		values := []any{"on", true, 2147483647, int64(9223372036854775807), 16777217, int64(9007199254740993)}
		others := []any{"off", false, -2147483648, int64(-9223372036854775808), 16777219, int64(9007199254740995)}
		i := int(backing % 6)
		member := map[string]any{"type": kinds[i], "enum": []any{values[i], others[i]}, "default": others[i]}
		if formats[i] != "" {
			member["format"] = formats[i]
		}
		literal := map[string]any{"version": "v1", "kind": "a", "pin": values[i]}
		if bad {
			delete(literal, "pin")
		}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{
			"U": map[string]any{"allOf": []any{map[string]any{"required": []string{"version"}, "properties": map[string]any{"version": map[string]any{"const": "v1"}}}, map[string]any{"oneOf": []any{
				map[string]any{"required": []string{"kind"}, "properties": map[string]any{"kind": map[string]any{"const": "a"}, "pin": member}},
				map[string]any{"required": []string{"kind"}, "properties": map[string]any{"kind": map[string]any{"const": "b"}, "other": map[string]any{"type": "string"}}},
			}}}},
			"Holder": map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/U", "default": literal}}},
		}}}
		input, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		if bad {
			if err == nil || len(first.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/value/default" {
				t.Fatal("missing pin accepted or lost Pointer", err)
			}
			return
		}
		if err != nil {
			t.Fatal("selected constructor rejected", i, err)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("composed defaults nondeterminism", err)
		}
	})
}
