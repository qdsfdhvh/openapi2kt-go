package core_test

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func TestBooleanRootNullableCatalogOrder(t *testing.T) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(booleanRootNullableInput(t), &top); err != nil {
		t.Fatal(err)
	}
	var components map[string]json.RawMessage
	if err := json.Unmarshal(top["components"], &components); err != nil {
		t.Fatal(err)
	}
	var schemas map[string]json.RawMessage
	if err := json.Unmarshal(components["schemas"], &schemas); err != nil {
		t.Fatal(err)
	}
	var reversed bytes.Buffer
	reversed.WriteByte('{')
	for i, name := range []string{"ZAny", "Holder", "DNullableAgain", "CFalse", "BInherited", "ANullable"} {
		if i > 0 {
			reversed.WriteByte(',')
		}
		key, _ := json.Marshal(name)
		reversed.Write(key)
		reversed.WriteByte(':')
		reversed.Write(schemas[name])
	}
	reversed.WriteByte('}')
	components["schemas"] = reversed.Bytes()
	top["components"], _ = json.Marshal(components)
	input, _ := json.Marshal(top)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(booleanRootNullableInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			before, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			r.Document = input
			after, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal(style, checked, "catalog spelling order", err)
			}
		}
	}
}
func TestBooleanRootNullableCheckedEdges(t *testing.T) {
	for _, edges := range []int{16, 17} {
		var leaf any = map[string]any{"$ref": "#/components/schemas/Nullable"}
		for i := 0; i < edges; i++ {
			leaf = map[string]any{"type": "array", "minItems": 0, "items": leaf}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"ZAny": true, "Nullable": map[string]any{"$ref": "#/components/schemas/ZAny", "nullable": true}, "Holder": map[string]any{"properties": map[string]any{"value": leaf}}}}})
		r := request(input, "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(edges, got, err)
		}
	}
}
