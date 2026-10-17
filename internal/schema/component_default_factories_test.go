package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

// A component's own root default attaches canonical JSON to its Model and never
// touches the property list, presence classification or nested declarations.
func TestComponentDefaultFactoryAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"Simple":{"type":"object","required":["z"],"properties":{"z":{"type":"string"},"n":{"type":"integer","default":7}},"default":{"z":"1"}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	model := componentModel(plan.Models, "SimpleDto")
	if model == nil {
		t.Fatal("component model not planned")
	}
	if model.ComponentDefault != `{"z":"1"}` {
		t.Fatalf("lost canonical component default: %q", model.ComponentDefault)
	}
	if len(model.Properties) != 2 {
		t.Fatalf("component default changed the property list: %+v", model.Properties)
	}
	// A required property keeps no initializer; the optional sibling keeps its
	// own field default.
	if z := model.Properties[0]; z.Default != "" || z.DefaultNull {
		t.Fatalf("component default gave the required property an initializer: %+v", z)
	}
	if n := model.Properties[1]; n.Default == "" {
		t.Fatalf("component default dropped the optional sibling's own default: %+v", n)
	}
	// A component without the annotation keeps an empty marker.
	plain, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	if got := componentModel(plain.Models, "SDto"); got == nil || got.ComponentDefault != "" {
		t.Fatalf("annotation-less component gained a factory: %+v", got)
	}
}

// Non-model roles keep their existing rejection at the original annotation
// Pointer; the factory contract adds no new shape grammar.
func TestComponentDefaultFactoryExclusions(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"empty raw object", `{"S":{"type":"object","default":{}}}`},
		{"reference wrapper", `{"T":{"type":"object","properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}`},
		{"inline union", `{"S":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"x"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto"}); err == nil {
				t.Fatal("excluded component role admitted")
			}
		})
	}
}

// A bad annotation keeps the original `.../default/...` Pointer.
func TestComponentDefaultFactoryDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, schemas, pointer string }{
		{"missing required", `{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{}}}`, "/components/schemas/S/default"},
		{"wrong leaf", `{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{"z":false}}}`, "/components/schemas/S/default/z"},
		{"unknown key", `{"S":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":{"extra":1}}}`, "/components/schemas/S/default/extra"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
			if err == nil || err.Pointer == nil || *err.Pointer != tc.pointer {
				t.Fatalf("want %s, got %v", tc.pointer, err)
			}
		})
	}
}

func loadComponentDocument(t *testing.T, schemas string) *document.Value {
	t.Helper()
	root, err := document.Load([]byte(`{"openapi":"3.1.0","components":{"schemas":`+schemas+`}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// Scalar and String-enum components attach the canonical annotation to their
// alias declaration without changing the declared type.
func TestComponentDefaultFactoryScalarAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"Name":{"type":"string","default":"x"},"Mode":{"type":"string","enum":["a","b"],"default":"a"}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	name := componentDeclaration(plan.Models, "NameDto")
	alias, ok := name.(*ScalarAlias)
	if !ok || alias.Scalar != "kotlin.String" || alias.ComponentDefault != `"x"` {
		t.Fatalf("lost scalar alias or factory: %+v", name)
	}
	mode := componentDeclaration(plan.Models, "ModeDto")
	enum, ok := mode.(*StringEnum)
	if !ok || len(enum.Members) != 2 || enum.ComponentDefault != `"a"` {
		t.Fatalf("lost enum members or factory: %+v", mode)
	}
}

// A scalar component factory never widens admission for other scalar roots.
func TestComponentDefaultFactoryScalarExclusions(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"bytes", `{"S":{"type":"string","format":"byte","default":"x"}}`},
		{"caller native", `{"S":{"type":"string","x-native-type":"org.example.Money","default":"x"}}`},
		{"nullable scalar", `{"S":{"type":"string","nullable":true,"default":"x"}}`},
		{"const", `{"S":{"type":"string","const":"x","default":"x"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto"}); err == nil {
				t.Fatal("excluded scalar root admitted")
			}
		})
	}
}

// An allOf component attaches the canonical annotation to its Model without
// touching the merged shape; a `$ref` wrapper component stays excluded.
func TestComponentDefaultFactoryAllOfAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"S":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"default":{"a":"x"}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	model := componentModel(plan.Models, "SDto")
	if model == nil || model.ComponentDefault != `{"a":"x"}` {
		t.Fatalf("lost allOf component default: %+v", model)
	}
	if len(model.Properties) != 2 {
		t.Fatalf("allOf component default changed the property list: %+v", model.Properties)
	}
	if _, _, err := Analyze(loadComponentDocument(t, `{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"}); err == nil {
		t.Fatal("$ref wrapper component admitted")
	}
}

// A single-layer collection component attaches the canonical annotation to its
// ValueAlias without changing the declared element type.
func TestComponentDefaultFactoryCollectionAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"Names":{"type":"array","items":{"type":"string"},"default":["a"]},"Bags":{"type":"object","additionalProperties":{"type":"integer"},"default":{"k":1}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	names := componentDeclaration(plan.Models, "NamesDto")
	alias, ok := names.(*ValueAlias)
	if !ok || alias.ComponentDefault != `["a"]` {
		t.Fatalf("lost List alias or factory: %+v", names)
	}
	if alias.Type.Kind != List || alias.Type.Element == nil {
		t.Fatalf("collection alias shape changed: %+v", alias.Type)
	}
	bags := componentDeclaration(plan.Models, "BagsDto")
	bag, ok := bags.(*ValueAlias)
	if !ok || bag.ComponentDefault != `{"k":1}` {
		t.Fatalf("lost Map alias or factory: %+v", bags)
	}
}

// Collection roles the field walk already excludes stay excluded at the root.
func TestComponentDefaultFactoryCollectionExclusionAuthority(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"nested container", `{"S":{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[["a"]]}}`},
		{"allOf wrapped", `{"S":{"allOf":[{"type":"array","items":{"type":"string"}}],"default":["a"]}}`},
		{"ref element", `{"T":{"type":"string"},"S":{"type":"array","items":{"$ref":"#/components/schemas/T"},"default":["a"]}}`},
		{"nullable root", `{"S":{"type":"array","nullable":true,"items":{"type":"string"},"default":null}}`},
		{"bad element", `{"S":{"type":"array","items":{"type":"integer"},"default":["a"]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}); err == nil {
				t.Fatal("excluded collection role admitted")
			}
		})
	}
}

// A temporal component attaches the canonical lexeme to its ValueAlias and keeps
// the actual Instant/LocalDate declaration.
func TestComponentDefaultFactoryTemporalAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"When":{"type":"string","format":"date-time","default":"2024-01-02T03:04:05Z"},"Day":{"type":"string","format":"date","default":"2024-02-29"}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	when := componentDeclaration(plan.Models, "WhenDto")
	alias, ok := when.(*ScalarAlias)
	if !ok || alias.ComponentDefault != `"2024-01-02T03:04:05Z"` || alias.Scalar != "kotlin.time.Instant" {
		t.Fatalf("lost Instant alias or factory: %+v", when)
	}
	day := componentDeclaration(plan.Models, "DayDto")
	dayAlias, ok := day.(*ScalarAlias)
	if !ok || dayAlias.ComponentDefault != `"2024-02-29"` || dayAlias.Scalar != "kotlinx.datetime.LocalDate" {
		t.Fatalf("lost LocalDate alias or factory: %+v", day)
	}
}

// A bad lexeme and a non-temporal format keep their existing rejection.
func TestComponentDefaultFactoryTemporalExclusions(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"bad date lexeme", `{"S":{"type":"string","format":"date","default":"2023-02-29"}}`},
		{"bad instant lexeme", `{"S":{"type":"string","format":"date-time","default":"nope"}}`},
		{"bytes format", `{"S":{"type":"string","format":"byte","default":"AAEC"}}`},
		{"nullable temporal", `{"S":{"type":"string","nullable":true,"format":"date","default":null}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}); err == nil {
				t.Fatal("excluded temporal root admitted")
			}
		})
	}
}

// A native enum component attaches the canonical annotation to its ScalarAlias
// and keeps the scalar wire type.
func TestComponentDefaultFactoryNativeEnumAuthority(t *testing.T) {
	plan, warnings, err := Analyze(loadComponentDocument(t, `{"Code":{"type":"integer","format":"int32","enum":[1,2],"default":1},"Flag":{"type":"boolean","enum":[true,false],"default":true}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, scalar string }{{"CodeDto", "kotlin.Int"}, {"FlagDto", "kotlin.Boolean"}} {
		declaration := componentDeclaration(plan.Models, tc.name)
		alias, ok := declaration.(*ScalarAlias)
		if !ok || alias.Scalar != tc.scalar || alias.ComponentDefault == "" {
			t.Fatalf("%s: lost native scalar alias or factory: %+v", tc.name, declaration)
		}
	}
	// The approximation warning is reported once per component.
	count := 0
	for _, warning := range warnings {
		if warning.Code == "W_ENUM_NATIVE_APPROXIMATION" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("want one native-enum warning per component, got %d: %v", count, warnings)
	}
}

// An unformatted numeric enum component keeps its existing rejection; a
// non-member value is rejected before rounding.
func TestComponentDefaultFactoryNativeEnumExclusions(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"unformatted numeric enum", `{"S":{"type":"integer","enum":[1],"default":1}}`},
		{"native non-member", `{"S":{"type":"integer","format":"int32","enum":[1,2],"default":9}}`},
		{"boolean non-member", `{"S":{"type":"boolean","enum":[true],"default":false}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}); err == nil {
				t.Fatal("excluded native enum role admitted")
			}
		})
	}
}
