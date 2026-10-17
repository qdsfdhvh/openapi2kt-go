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

// Scalar and String-enum components own their value outright, so their factory
// returns the alias type instead of a constructor. A typealias cannot own
// members, so the scalar form is a top-level function.
func TestComponentDefaultFactoryScalarAndEnum(t *testing.T) {
	const document = `{"Name":{"type":"string","default":"x"},"Count":{"type":"integer","default":5},"Ratio":{"type":"number","default":1.5},"Flag":{"type":"boolean","default":true},"Mode":{"type":"string","enum":["a","b"],"default":"a"}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(schemas(document), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			for _, marker := range []struct{ file, want string }{
				{"NameDto", "public fun `NameDto default`(): NameDto ="},
				{"CountDto", "public fun `CountDto default`(): CountDto ="},
				{"RatioDto", "public fun `RatioDto default`(): RatioDto ="},
				{"FlagDto", "public fun `FlagDto default`(): FlagDto ="},
				{"ModeDto", "public fun `ModeDto default`(): ModeDto ="},
			} {
				if text := source(t, got, marker.file); !strings.Contains(text, marker.want) {
					t.Fatal(marker.file, text)
				}
			}
			// The alias and enum declarations themselves are unchanged.
			if text := source(t, got, "NameDto"); !strings.Contains(text, "public typealias NameDto = kotlin.String\n") {
				t.Fatal("scalar alias changed", text)
			}
		}
	}
}

// The scalar/enum factory keeps the original annotation Pointer and admits only
// the plain scalar kinds whose field role exists.
func TestComponentDefaultFactoryScalarGuards(t *testing.T) {
	for _, tc := range []struct{ name, schemas, pointer string }{
		{"wrong kind", `{"S":{"type":"string","default":5}}`, "/components/schemas/S/default"},
		{"enum non-member", `{"S":{"type":"string","enum":["a"],"default":"z"}}`, "/components/schemas/S/default"},
		{"integer non-integral", `{"S":{"type":"integer","default":1.5}}`, "/components/schemas/S/default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatalf("want %s, got %v", tc.pointer, err)
			}
		})
	}
	// Native markers, temporal, bytes and value-class roots keep their roles.
	for _, tc := range []struct{ name, schemas string }{
		{"temporal", `{"S":{"type":"string","format":"date-time","default":"2024-01-02T03:04:05Z"}}`},
		{"native enum", `{"S":{"type":"integer","format":"int32","enum":[1,2],"default":1}}`},
		{"bytes", `{"S":{"type":"string","format":"byte","default":"x"}}`},
		{"caller native", `{"S":{"type":"string","x-native-type":"org.example.Money","default":"x"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("excluded scalar root admitted: %v", err)
			}
		})
	}
}

// An allOf component's own root default emits the factory, mirroring the
// consuming-field allOf walk. A `$ref` wrapper component stays excluded.
func TestComponentDefaultFactoryAllOf(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"S":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"default":{"a":"x"}}}}}`)
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			if text := source(t, got, "SDto"); !strings.Contains(text, "public fun default(): SDto =") {
				t.Fatal(checked, style, text)
			}
		}
	}
	// The wrapper component keeps its rejection.
	wrapper := []byte(`{"openapi":"3.1.0","components":{"schemas":{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}}}`)
	r2 := request(wrapper, "Dto")
	r2.Options.Validation = true
	report, err2 := core.GenerateWithDiagnostics(r2)
	ds, ok := err2.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/S/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("$ref wrapper admitted: %v", err2)
	}
}
