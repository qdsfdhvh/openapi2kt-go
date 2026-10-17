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
		{"collection alias", `{"S":{"type":"array","items":{"type":"string"},"default":["a"]}}`},
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
		{"temporal", `{"S":{"type":"string","format":"date-time","default":"2024-01-02T03:04:05Z"}}`},
		{"bytes", `{"S":{"type":"string","format":"byte","default":"x"}}`},
		{"caller native", `{"S":{"type":"string","x-native-type":"org.example.Money","default":"x"}}`},
		{"native enum", `{"S":{"type":"integer","format":"int32","enum":[1,2],"default":1}}`},
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
