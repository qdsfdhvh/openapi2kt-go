package core_test

import (
	"errors"
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
	// Native markers and value-class roots keep their roles. A bytes root is now
	// admitted, so a non-array bytes value fails as an invalid annotation value
	// rather than as an unsupported shape.
	for _, tc := range []struct{ name, schemas, code string }{
		{"bytes wrong value", `{"S":{"type":"string","format":"byte","default":"x"}}`, "E_SCHEMA_INVALID"},
		{"caller native", `{"S":{"type":"string","x-native-type":"org.example.Money","default":"x"}}`, "E_SCHEMA_UNSUPPORTED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 || len(got.Artifacts.Files) != 0 {
				t.Fatalf("excluded scalar root admitted: %v", err)
			}
			if string(ds[0].Code) != tc.code {
				t.Fatalf("want %s, got %s", tc.code, ds[0].Code)
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
	// A `$ref` wrapper now owns its own factory on the alias, while the target
	// gains nothing. Graduated from the previous rejection of the whole shape.
	wrapper := []byte(`{"openapi":"3.1.0","components":{"schemas":{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}}}`)
	r2 := request(wrapper, "Dto")
	r2.Options.Validation = true
	report2, err2 := core.GenerateWithDiagnostics(r2)
	if err2 != nil {
		t.Fatalf("$ref wrapper lost its factory: %v", err2)
	}
	wrapperSource := source(t, report2, "SDto")
	if !strings.Contains(wrapperSource, "public fun `SDto default`(): SDto =") {
		t.Fatalf("wrapper factory missing:\n%s", wrapperSource)
	}
	if strings.Contains(source(t, report2, "TDto"), "`TDto default`") {
		t.Fatalf("wrapper default leaked onto the target")
	}
}

// A single-layer List/typed Map component root may own its root default and
// emits a top-level factory, since a typealias cannot carry members.
func TestComponentDefaultFactoryCollections(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Names":{"type":"array","items":{"type":"string"},"default":["a","b"]},"Counts":{"type":"array","items":{"type":"integer"},"default":[1,2]},"Bags":{"type":"object","additionalProperties":{"type":"integer"},"default":{"k":1}},"Dates":{"type":"array","items":{"type":"string","format":"date"},"default":["2024-02-29"]},"Codes":{"type":"array","items":{"type":"string","enum":["one","two"]},"default":["one"]}}}}`), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			for _, file := range []string{"NamesDto", "CountsDto", "BagsDto", "DatesDto", "CodesDto"} {
				if text := source(t, got, file); !strings.Contains(text, "public fun `"+file+" default`(): "+file+" =") {
					t.Fatal(file, text)
				}
			}
			if text := source(t, got, "NamesDto"); !strings.Contains(text, "public typealias NamesDto = kotlin.collections.List<kotlin.String>") {
				t.Fatal("collection alias changed", text)
			}
		}
	}
}

// Nested containers, allOf-wrapped containers, reference elements, nullable
// roots and bad elements keep the exclusions the field role already enforces.
func TestComponentDefaultFactoryCollectionExclusions(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"nested container", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[["a"]]}}}}`},
		{"allOf wrapped", `{"openapi":"3.1.0","components":{"schemas":{"S":{"allOf":[{"type":"array","items":{"type":"string"}}],"default":["a"]}}}}`},
		{"ref element", `{"openapi":"3.1.0","components":{"schemas":{"T":{"type":"string"},"S":{"type":"array","items":{"$ref":"#/components/schemas/T"},"default":["a"]}}}}`},
		{"nullable root", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"array","nullable":true,"items":{"type":"string"},"default":null}}}}`},
		{"bad element", `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"array","items":{"type":"integer"},"default":["a"]}}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request([]byte(tc.input), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("excluded collection role admitted: %v", err)
			}
		})
	}
}

// A standard temporal scalar component root may own its root default. The
// factory returns the actual Instant/LocalDate alias and validates the lexeme
// through the existing temporal role.
func TestComponentDefaultFactoryTemporal(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"When":{"type":"string","format":"date-time","default":"2024-01-02T03:04:05Z"},"Day":{"type":"string","format":"date","default":"2024-02-29"}}}}`), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			if text := source(t, got, "WhenDto"); !strings.Contains(text, "public typealias WhenDto = kotlin.time.Instant") ||
				!strings.Contains(text, "public fun `WhenDto default`(): WhenDto =") {
				t.Fatal("Instant factory", text)
			}
			if text := source(t, got, "DayDto"); !strings.Contains(text, "public typealias DayDto = kotlinx.datetime.LocalDate") ||
				!strings.Contains(text, "public fun `DayDto default`(): DayDto =") {
				t.Fatal("LocalDate factory", text)
			}
		}
	}
	// A bad lexeme keeps its rejection and must report the component's own root
	// pointer, not a bare `/default`.
	for _, tc := range []struct{ name, schemas string }{
		{"bad date lexeme", `{"S":{"type":"string","format":"date","default":"2023-02-29"}}`},
		{"bad instant lexeme", `{"S":{"type":"string","format":"date-time","default":"nope"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 || len(got.Artifacts.Files) != 0 {
				t.Fatalf("excluded temporal root admitted: %v", err)
			}
			if ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/S/default" {
				t.Fatalf("bad temporal default reported %v, want the component root", ds[0].Pointer)
			}
		})
	}
}

// A numeric/Boolean native enum component root may own its root default. The
// scalar wire type, the original exact vocabulary membership and the existing
// approximation warning are retained.
func TestComponentDefaultFactoryNativeEnum(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","format":"int32","enum":[1,2,3],"default":1},"Wide":{"type":"integer","format":"int64","enum":[10,20],"default":10},"Flag":{"type":"boolean","enum":[true,false],"default":true},"Ratio":{"type":"number","format":"double","enum":[1.5,2.5],"default":1.5}}}}`), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			for _, marker := range []struct{ file, want string }{
				{"CodeDto", "public typealias CodeDto = kotlin.Int"},
				{"WideDto", "public typealias WideDto = kotlin.Long"},
				{"FlagDto", "public typealias FlagDto = kotlin.Boolean"},
				{"RatioDto", "public typealias RatioDto = kotlin.Double"},
			} {
				if text := source(t, got, marker.file); !strings.Contains(text, marker.want) ||
					!strings.Contains(text, "public fun `"+marker.file+" default`(): "+marker.file+" =") {
					t.Fatal(marker.file, text)
				}
			}
			// The native approximation warning stays.
			found := false
			for _, w := range got.Warnings {
				if w.Code == "W_ENUM_NATIVE_APPROXIMATION" {
					found = true
				}
			}
			if !found {
				t.Fatal("native enum approximation warning lost", got.Warnings)
			}
		}
	}
	// A non-member value keeps the exact pre-rounding membership check.
	for _, tc := range []struct{ name, schemas string }{
		{"native non-member", `{"S":{"type":"integer","format":"int32","enum":[1,2],"default":9}}`},
		{"boolean non-member", `{"S":{"type":"boolean","enum":[true],"default":false}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("native non-member admitted: %v", err)
			}
		})
	}
}

// A nullable ordinary object component may own a null root default. The factory
// returns the nullable type directly and needs no decode helper.
func TestComponentDefaultFactoryNullableObject(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"MaybeRec":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":null}}}}`), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "MaybeRecDto")
			if !strings.Contains(text, "public fun default(): MaybeRecDto? = null") {
				t.Fatal("nullable factory", text)
			}
			// A null literal must not go through the non-null component serializer.
			if strings.Contains(text, "openapi-kt component default") {
				t.Fatal("null default kept a decode helper", text)
			}
		}
	}
	// A nullable root whose annotation is not null keeps its existing rejection.
	for _, tc := range []struct{ name, schemas string }{
		{"non-null annotation on nullable root", `{"S":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":{"z":"1"}}}`},
		{"nullable root with counts", `{"S":{"type":"object","nullable":true,"minProperties":0,"properties":{"z":{"type":"string"}},"default":null}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.schemas), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("excluded nullable role admitted: %v", err)
			}
		})
	}
}
