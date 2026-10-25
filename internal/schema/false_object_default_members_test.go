package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseObjectActualFactoryAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Holder":{"required":["description"],"properties":{"description":{"$ref":"#/components/schemas/Blocked","nullable":true}},"default":{"description":null}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falseobjects", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if model == nil || model.ComponentDefault != `{"description":null}` {
		t.Fatal("missing actual object factory")
	}
	saved := model.ComponentDefault
	model.ComponentDefault = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("removed actual false object factory authorized")
	}
	model.ComponentDefault = saved
	proof := a.falseObjectFactories[model]
	delete(a.falseObjectFactories, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing actual model factory proof authorized")
	}
	a.falseObjectFactories[model] = proof
	model.ComponentDefault = `{}`
	if a.validateInternalModels(ds) == nil {
		t.Error("changed canonical literal initializer authorized")
	}
	model.ComponentDefault = saved
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	defaultValue := get(a.components["Holder"].Schema, "default")
	copyValue := get(get(get(get(copied, "components"), "schemas"), "Holder"), "default")
	for i, m := range defaultValue.Members() {
		defaultValue.Members()[i].Value = get(copyValue, m.Name)
		if a.validateInternalModels(ds) == nil {
			t.Error("same-byte metadata-named default literal replaced")
		}
		defaultValue.Members()[i] = m
	}

	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
