package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestUnformattedNativeFactoryOriginalSourcePermission(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"type":"integer","enum":[1,2],"default":1},"Alias":{"$ref":"#/components/schemas/T","default":2}}`)
	before, _ := root.MarshalJSON()
	plan, _, err := Analyze(root, Options{PackageName: "org.example.native", Suffix: "Dto", Validation: true})
	if err != nil {
		t.Fatal(err)
	}
	target, ok := componentDeclaration(plan.Models, "TDto").(*ScalarAlias)
	if !ok || target.Scalar != "kotlin.Int" || target.ComponentDefault != "1" {
		t.Fatal("lost actual native root mapping", target)
	}
	alias, ok := componentDeclaration(plan.Models, "AliasDto").(*ValueAlias)
	if !ok || alias.ComponentDefault != "2" {
		t.Fatal("independent alias factory inherited target value", alias)
	}
	source := get(get(get(root, "components"), "schemas"), "T")
	a := analyzer{options: Options{Validation: true}, componentDefaultSchemas: map[*document.Value]bool{source: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
	if scalar, failure := a.nativeEnum("integer", source, "/components/schemas/T"); failure != nil || scalar != "kotlin.Int" {
		t.Fatal("original marked root rejected", failure)
	}
	copyRoot := loadComponentDocument(t, `{"T":{"type":"integer","enum":[1,2],"default":1}}`)
	copied := get(get(get(copyRoot, "components"), "schemas"), "T")
	if _, failure := a.nativeEnum("integer", copied, "/components/schemas/T"); failure == nil {
		t.Fatal("same-byte copy gained root own-default permission")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("native factory changed original AST")
	}
}
