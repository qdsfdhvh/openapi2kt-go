package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestUUIDFactorySourceAndPermission(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"string","format":"uuid","default":"ordinary text"},"M":{"$ref":"#/components/schemas/T","default":"middle"},"N":{"$ref":"#/components/schemas/M","default":"outer"}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	target, ok := componentDeclaration(plan.Models, "TDto").(*ScalarAlias)
	if !ok || target.Scalar != "kotlin.String" || target.ComponentDefault != `"ordinary text"` {
		t.Fatal("UUID root lost String semantics", target)
	}
	outer, ok := componentDeclaration(plan.Models, "NDto").(*ScalarAlias)
	if !ok || outer.Scalar != "kotlin.String" || outer.ComponentDefault != `"outer"` {
		t.Fatal("UUID alias lost String mapping or own annotation", outer)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("UUID factory mutated original sources")
	}
	source := get(get(get(root, "components"), "schemas"), "T")
	a := &analyzer{componentDefaultSchemas: map[*document.Value]bool{source: true}}
	if scalar, err := scalarTypeFor(a, "string", source, "/components/schemas/T"); err != nil || scalar != "kotlin.String" {
		t.Fatal("original UUID source denied", scalar, err)
	}
	clone := loadComponentDocument(t, `{"T":{"type":"string","format":"uuid","default":"ordinary text"}}`)
	copied := get(get(get(clone, "components"), "schemas"), "T")
	if _, err := scalarTypeFor(a, "string", copied, "/components/schemas/T"); err == nil || err.Code != "E_SCHEMA_UNSUPPORTED" {
		t.Fatal("UUID default permission leaked to copied source", err)
	}
}

func TestNullableUUIDFactorySourceAndIntent(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":["string","null"],"format":"uuid","default":null},"M":{"$ref":"#/components/schemas/T","nullable":false,"default":"outer"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, value string
		nullable    bool
	}{
		{"TDto", "null", true}, {"MDto", `"outer"`, false}, {"NDto", "null", true},
	} {
		alias, ok := componentDeclaration(plan.Models, tc.name).(*ValueAlias)
		if !ok || (alias.Type.Kind == Nullable) != tc.nullable || bare(alias.Type).Name != "kotlin.String" || alias.ComponentDefault != tc.value {
			t.Fatal("UUID alias lost actual String codec/nullable intent/default ownership", tc.name, alias)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("nullable UUID factory mutated original sources")
	}
}
