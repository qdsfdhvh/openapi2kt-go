package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const componentDefaultDocument = `{"Simple":{"type":"object","required":["z"],"properties":{"z":{"type":"string"},"n":{"type":"integer","default":7}},"default":{"z":"1"}},"Optional":{"type":"object","properties":{"z":{"type":"string"}},"default":{"z":"opt"}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/Simple"}}}}`

// A component's own root default emits an opt-in factory and never an
// initializer, so the constructor, presence classification and serializer keep
// their existing bytes.
func TestComponentDefaultFactories(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			input := schemas(componentDefaultDocument)
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			simple := source(t, got, "SimpleDto")
			for _, marker := range []string{
				"public fun default(): SimpleDto =",
				"`openapi-kt component default ",
				`("{\"z\":\"1\"}")`,
			} {
				if !strings.Contains(simple, marker) {
					t.Fatal(checked, style, "lost factory", marker, simple)
				}
			}
			// The required property keeps its constructor argument and gains no
			// initializer; the optional sibling keeps its own field default.
			if !strings.Contains(simple, "public val z: kotlin.String,") || strings.Contains(simple, "public val z: kotlin.String =") {
				t.Fatal("required property acquired an initializer", simple)
			}
			if !strings.Contains(simple, "public val n: kotlin.Int? = 7,") {
				t.Fatal("own property default lost", simple)
			}
			// A consumer does not inherit the component default.
			consumer := source(t, got, "ConsumerDto")
			if strings.Contains(consumer, "fun default()") || !strings.Contains(consumer, ".SimpleDto? = null,") {
				t.Fatal("component default leaked into a consumer", consumer)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(again.Artifacts, got.Artifacts) {
				t.Fatal("nondeterministic component default output", err)
			}
		}
	}
}

func TestComponentDefaultFactoryGuards(t *testing.T) {
	for _, tc := range []struct{ input, pointer string }{
		{`{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{}}}}}`, "/components/schemas/S/default"},
		{`{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{"z":false}}}}}`, "/components/schemas/S/default/z"},
		{`{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{"extra":1}}}}}`, "/components/schemas/S/default/extra"},
	} {
		r := request([]byte(tc.input), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatalf("%s: %v", tc.pointer, err)
		}
	}
	// Roles outside this contract keep their existing rejection location.
	for _, tc := range []struct{ name, input, pointer string }{
		{"empty raw object", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"object","default":{}}}}}`, "/components/schemas/S/default"},
		{"string enum", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"string","enum":["a"],"default":"a"}}}}`, "/components/schemas/S/default"},
		{"scalar", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"string","default":"x"}}}}`, "/components/schemas/S/default"},
		{"collection alias", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"array","items":{"type":"string"},"default":["a"]}}}}`, "/components/schemas/S/default"},
		{"reference wrapper", `{"openapi":"3.1.0","components":{"schemas":{"T":{"type":"object","properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}}}`, "/components/schemas/S/default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request([]byte(tc.input), "Dto")
			r.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatalf("excluded role admitted: %v", err)
			}
		})
	}
}
