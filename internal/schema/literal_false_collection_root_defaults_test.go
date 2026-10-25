package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestLiteralFalseCollectionRootFactoryActualAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"values":{"additionalProperties":{"type":"array","nullable":true,"items":false}}},"default":{"values":{"description":null}}}}}}`)
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
	if !ok || !a.literalFalseObjectDefaultUses[model] || factory.Owner != a.components["Holder"] || factory.Encoded != "{\"values\":{\"description\":null}}" {
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
	values := get(get(schema, "default"), "values")
	for i, member := range values.Members() {
		// Annotation-looking keys inside JSON values are still literal data;
		// replacing an equal null node must invalidate the captured proof.
		values.Members()[i].Value = get(get(copied, "values"), member.Name)
		if a.validateInternalModels(ds) == nil {
			t.Error("same-text metadata-named literal data substitution authorized")
		}
		values.Members()[i] = member
	}
	origin := a.literalFalseFieldOrigins[model]
	field := origin.Fields[0].Plan.Collection
	validation := a
	validation.componentDefaultCountRoot = schema
	if node, err := validation.literalFalseObjectDefaultNode(field.Leaf, field.Pointer, field.Height+1); err != nil || node == nil || !node.uninhabited || node.ty.Kind != Named {
		t.Fatal("original collection leaf lost actual factory codec", node, err)
	}
	if node, err := validation.literalFalseObjectDefaultNode(field.Leaf, field.Pointer+"/unrelated", field.Height+1); err != nil || node != nil {
		t.Fatal("original leaf borrowed at unrelated Pointer", node, err)
	}
	validation.componentDefaultCountRoot = nil
	validation.falseDTOFieldDefaultScope = &falseDTOFieldDefaultScope{Target: model, TargetSchema: schema}
	if node, err := validation.literalFalseObjectDefaultNode(field.Leaf, field.Pointer, field.Height+1); err != nil || node != nil {
		t.Fatal("owner-only collection default granted consuming field role", node, err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
