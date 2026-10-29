package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzMetadataStringEnumModels(f *testing.F) {
	f.Add("ready", uint8(0), false)
	f.Add("null", uint8(3), true)
	f.Add("future", uint8(15), true)
	f.Add("ready", uint8(16), false)
	f.Add("future", uint8(31), true)
	f.Fuzz(func(t *testing.T, wire string, mode uint8, checked bool) {
		if len(wire) > 96 {
			t.Skip()
		}
		var leaf map[string]any
		if err := json.Unmarshal([]byte(`{`+tupleStringMetadata[int(mode)%len(tupleStringMetadata)]+`}`), &leaf); err != nil {
			t.Fatal(err)
		}
		leaf["type"], leaf["enum"], leaf["default"] = "string", []any{wire}, wire
		var value any = wire
		field := map[string]any{"$ref": "#/components/schemas/Maybe"}
		if mode&1 != 0 {
			field = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Maybe"}}
			value = []any{wire, nil}
		}
		invalid := mode&8 != 0
		if invalid {
			value = false
		}
		field["default"] = value
		holder := map[string]any{"properties": map[string]any{"value": field}}
		if mode&2 != 0 {
			holder["required"] = []string{"value"}
		}
		components := map[string]any{"Value": leaf, "Maybe": map[string]any{"$ref": "#/components/schemas/Value", "nullable": true, "default": nil}, "Holder": holder}
		if mode&16 != 0 {
			delete(components, "Value")
			components["Owner"] = map[string]any{"properties": map[string]any{"state": leaf}}
			components["Maybe"].(map[string]any)["$ref"] = "#/components/schemas/Owner/properties/state"
		}
		data, _ := json.Marshal(components)
		rq := request(schemas(string(data)), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		again, repeated := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(got, again) || !reflect.DeepEqual(err, repeated) {
			t.Fatal("metadata enum model nondeterminism")
		}
		if (err != nil) != invalid {
			t.Fatal("metadata enum independent annotation contract", mode, err)
		}
		if err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial metadata enum model output")
		}
	})
}
