package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func TestBooleanRootNullableNoDropAndLegacy(t *testing.T) {
	for _, input := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":"true"},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":true,"unknown":true},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":true},"ZAny":false}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/Missing","nullable":true},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":true},"Holder":{"allOf":[{"$ref":"#/components/schemas/Alias"}]},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":false},"Holder":{"allOf":[{"$ref":"#/components/schemas/Alias"}]},"ZAny":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":true},"Holder":{"properties":{"v":true,"use":{"$ref":"#/components/schemas/Holder/properties/v"}}},"ZAny":true}}}`,
		`{"openapi":"3.0.3","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZAny","nullable":true},"ZAny":true}}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request([]byte(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("excluded accepted/partial", checked, input, got, err)
			}
		}
	}
	for _, nullable := range []string{"", " ,\"nullable\":false"} {
		r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/ZInt"`+nullable+`},"ZInt":{"type":"integer"}}}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("default-off builtin alias widened", nullable, got, err)
		}
	}
}
func TestBooleanRootNullableIndependentLimits(t *testing.T) {
	for _, consumer := range []bool{false, true} {
		low := 48
		if consumer {
			low = 46
		}
		for _, edges := range []int{low, low + 1} {
			r := booleanRootChain(t, edges, consumer)
			var doc map[string]any
			if err := json.Unmarshal(r.Document, &doc); err != nil {
				t.Fatal(err)
			}
			schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
			for name, node := range schemas {
				if name != "ZAny" && name != "AHolder" {
					node.(map[string]any)["nullable"] = true
				}
			}
			r.Document, _ = json.Marshal(doc)
			for _, checked := range []bool{false, true} {
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if edges == low && err != nil || edges > low && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(consumer, edges, checked, got, err)
				}
			}
		}
	}
	for _, width := range []int{240, 241} {
		props := map[string]any{}
		required := []string{}
		for i := 0; i < width; i++ {
			name := fmt.Sprintf("v%03d", i)
			props[name] = map[string]any{"$ref": "#/components/schemas/Nullable"}
			required = append(required, name)
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"ZAny": true, "Nullable": map[string]any{"$ref": "#/components/schemas/ZAny", "nullable": true}, "Holder": map[string]any{"properties": props, "required": required}}}})
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, got, err)
		}
	}
}
