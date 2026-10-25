package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestLiteralFalseRootFactoryActualAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"blocked":false,"label":{"type":"string"}},"default":{"label":"seed"}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.literalfalse", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var model *Model
	for _, d := range ds {
		if m, ok := d.(*Model); ok && m.ClassName == "HolderDto" {
			model = m
		}
	}
	if model == nil {
		t.Fatal("missing actual owner")
	}
	factory, ok := a.literalFalseObjectFactories[model]
	if !ok || !a.literalFalseObjectDefaultUses[model] || factory.Owner != a.components["Holder"] || factory.Encoded != "{\"label\":\"seed\"}" {
		t.Fatal("missing actual factory proof")
	}
	for _, encoded := range []string{"", "{}", "{\"label\":\"other\"}"} {
		saved := model.ComponentDefault
		model.ComponentDefault = encoded
		if a.validateInternalModels(ds) == nil {
			t.Error("removed/changed actual factory authorized", encoded)
		}
		model.ComponentDefault = saved
	}
	delete(a.literalFalseObjectFactories, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing last factory proof authorized")
	}
	a.literalFalseObjectFactories[model] = factory
	delete(a.literalFalseObjectDefaultUses, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing own planning use authorized")
	}
	a.literalFalseObjectDefaultUses[model] = true
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copied := get(get(get(get(copyRoot, "components"), "schemas"), "Holder"), "default")
	schema := a.modelOrigins[model]
	for i, member := range schema.Members() {
		if member.Name != "default" {
			continue
		}
		schema.Members()[i].Value = copied
		if a.validateInternalModels(ds) == nil {
			t.Error("same-text source default substitution authorized")
		}
		schema.Members()[i] = member
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
