package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestEnumFormRetainsActualLocalDeclarationAndSourceTree(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","body":{"required":true,"content":{"application/x-www-form-urlencoded":{"schema":{"type":"object","properties":{"code":{"type":"string","enum":["ready","done"]},"count":{"type":"integer","enum":[1,2]}}}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.enumform", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	body, err := a.requestBody(get(root, "body"), "/body", "Submit")
	if err != nil {
		t.Fatal(err)
	}
	model, ok := body.Models[0].(*Model)
	if !ok || len(model.Nested) != 1 || len(body.FormFields) != 2 {
		t.Fatal("form lost actual retained model")
	}
	enum, ok := model.Nested[0].(*StringEnum)
	if !ok || body.FormFields[0].Type.WithoutRootNull().Name != enum.ClassName || &body.FormFields[0] != &model.Properties[0] {
		t.Fatal("form invented enum or property binding")
	}
	if model.Properties[1].Validation == nil || len(model.Properties[1].Validation.Enum) != 2 {
		t.Fatal("native membership checks lost")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("form rewrote original source")
	}
}
