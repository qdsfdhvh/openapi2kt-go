package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseReferenceOwnDefaultActualPropertyAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"required":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsefielddefaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var model *Model
	for _, d := range ds {
		if value, ok := d.(*Model); ok {
			model = value
		}
	}
	if model == nil || len(model.Properties) != 2 {
		t.Fatal("missing actual constructor")
	}
	if model.Properties[0].Default != "null" || !model.Properties[0].DefaultNull || model.Properties[1].Default != "" || model.Properties[1].DefaultNull {
		t.Fatal("own nullable default changed required/optional rules")
	}
	saved := model.Properties[0].Default
	model.Properties[0].Default = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("removed actual owned initializer authorized")
	}
	model.Properties[0].Default = saved
	model.Properties[1].DefaultNull = true
	if a.validateInternalModels(ds) == nil {
		t.Error("required actual parameter made optional")
	}
	model.Properties[1].DefaultNull = false
	model.Properties[1].Default = "null"
	if a.validateInternalModels(ds) == nil {
		t.Error("required actual parameter gained initializer")
	}
	model.Properties[1].Default = ""
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	original := get(get(a.components["Holder"].Schema, "properties"), "optional")
	copySchema := get(get(get(get(get(copied, "components"), "schemas"), "Holder"), "properties"), "optional")
	for i, m := range original.Members() {
		if m.Name == "default" {
			original.Members()[i].Value = get(copySchema, "default")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte owned default node replaced")
			}
			original.Members()[i] = m
		}
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
