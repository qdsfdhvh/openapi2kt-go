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
		{"const", `{"S":{"type":"string","const":"x","default":"x"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto"}); err == nil {
				t.Fatal("excluded scalar root admitted")
			}
		})
	}
	// A nullable scalar root supports both ordinary and null annotations.
	for _, checked := range []bool{false, true} {
		plan, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"string","nullable":true,"default":"x"}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if err != nil {
			t.Fatal(checked, err)
		}
		alias, ok := componentDeclaration(plan.Models, "SDto").(*ValueAlias)
		if !ok || alias.ComponentDefault != `"x"` {
			t.Fatalf("nullable scalar factory lost its value: %+v", componentDeclaration(plan.Models, "SDto"))
		}
		nullPlan, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"string","nullable":true,"default":null}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if err != nil {
			t.Fatal(checked, err)
		}
		if alias, ok := componentDeclaration(nullPlan.Models, "SDto").(*ValueAlias); !ok || alias.Type.Kind != Nullable || alias.ComponentDefault != "null" {
			t.Fatal("original nullable scalar null case lost", alias)
		}
	}
}

// An allOf component attaches the canonical annotation to its Model without
// touching the merged shape; a `$ref` wrapper attaches it to its own alias.
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
	// The wrapper owns the factory on its own alias; the target stays plain.
	wrapperPlan, _, err := Analyze(loadComponentDocument(t, `{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatalf("$ref wrapper lost its factory: %v", err)
	}
	alias, ok := componentDeclaration(wrapperPlan.Models, "SDto").(*ValueAlias)
	if !ok || alias.ComponentDefault != `{"z":"1"}` {
		t.Fatalf("wrapper did not own the annotation: %+v", componentDeclaration(wrapperPlan.Models, "SDto"))
	}
	if target, ok := componentDeclaration(wrapperPlan.Models, "TDto").(*Model); ok && target.ComponentDefault != "" {
		t.Fatalf("wrapper default leaked onto the target: %+v", target.ComponentDefault)
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
	// The exact original nullable-root refusal now owns a null factory.
	nullPlan, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"array","nullable":true,"items":{"type":"string"},"default":null}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	if alias, ok := componentDeclaration(nullPlan.Models, "SDto").(*ValueAlias); !ok || alias.Type.Kind != Nullable || alias.ComponentDefault != "null" {
		t.Fatal("original nullable List null case lost", alias)
	}
}

// Collection roles the field walk already excludes stay excluded at the root.
func TestComponentDefaultFactoryCollectionExclusionAuthority(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"nested container", `{"S":{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[["a"]]}}`},
		{"allOf wrapped", `{"S":{"allOf":[{"type":"array","items":{"type":"string"}}],"default":["a"]}}`},
		{"ref element", `{"T":{"type":"string"},"S":{"type":"array","items":{"$ref":"#/components/schemas/T"},"default":["a"]}}`},
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
	// Preserve the original nullable temporal refusal as a positive case.
	nullPlan, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"string","nullable":true,"format":"date","default":null}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	if alias, ok := componentDeclaration(nullPlan.Models, "SDto").(*ValueAlias); !ok || alias.Type.Kind != Nullable || alias.ComponentDefault != "null" {
		t.Fatal("original nullable date null case lost", alias)
	}
}

// A bad lexeme and a non-temporal format keep their existing rejection.
func TestComponentDefaultFactoryTemporalExclusions(t *testing.T) {
	for _, tc := range []struct{ name, schemas string }{
		{"bad date lexeme", `{"S":{"type":"string","format":"date","default":"2023-02-29"}}`},
		{"bad instant lexeme", `{"S":{"type":"string","format":"date-time","default":"nope"}}`},
		{"bytes format", `{"S":{"type":"string","format":"byte","default":"AAEC"}}`},
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

// A nullable ordinary object component attaches a null annotation and emits the
// factory without a decode helper.
func TestComponentDefaultFactoryNullableObjectAuthority(t *testing.T) {
	plan, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":null}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto"})
	if err != nil {
		t.Fatal(err)
	}
	model := componentModel(plan.Models, "SDto")
	if model == nil || model.ComponentDefault != "null" {
		t.Fatalf("lost nullable component default: %+v", model)
	}
	// The nullable-component-counts role stays excluded. A non-null annotation is
	// an ordinary value of the nullable object's declared type.
	for _, checked := range []bool{false, true} {
		nonNull, _, err := Analyze(loadComponentDocument(t, `{"S":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":{"z":"1"}}}`), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if err != nil {
			t.Fatal(checked, err)
		}
		if model := componentModel(nonNull.Models, "SDto"); model == nil || model.ComponentDefault != `{"z":"1"}` || !model.Nullable {
			t.Fatalf("nullable object non-null factory lost: %+v", model)
		}
	}
	for _, tc := range []struct{ name, schemas string }{
		{"counts", `{"S":{"type":"object","nullable":true,"minProperties":0,"properties":{"z":{"type":"string"}},"default":null}}`},
		{"counts non-null", `{"S":{"type":"object","nullable":true,"minProperties":0,"properties":{"z":{"type":"string"}},"default":{"z":"1"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Analyze(loadComponentDocument(t, tc.schemas), Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}); err == nil {
				t.Fatal("excluded nullable role admitted")
			}
		})
	}
}

// A nullable component root accepts a non-null annotation without changing its
// declared nullable type or its original source node.
func TestNullableComponentDefaultSourceAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"N":{"type":["string","null"],"default":"x"}}`)
	original, _ := get(get(get(root, "components"), "schemas"), "N").MarshalJSON()
	for _, checked := range []bool{false, true} {
		plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if err != nil {
			t.Fatal(checked, err)
		}
		declaration, ok := componentDeclaration(plan.Models, "NDto").(*ValueAlias)
		if !ok || declaration.ComponentDefault != `"x"` {
			t.Fatalf("nullable factory lost its canonical value: %+v", declaration)
		}
		if declaration.Type.Kind != Nullable {
			t.Fatalf("nullable declaration lost its public type: %+v", declaration.Type)
		}
		after, _ := get(get(get(root, "components"), "schemas"), "N").MarshalJSON()
		if string(original) != string(after) {
			t.Fatal("nullable planning rewrote the original schema node")
		}
	}
}
