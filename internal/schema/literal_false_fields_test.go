package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestLiteralFalseActualFieldAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"required":["required"],"properties":{"optional":false,"required":false}}}}}`)
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
		t.Fatal("missing actualowner")
	}
	origin := a.literalFalseFieldOrigins[model]
	owner := a.components["Holder"]
	ownerName := owner.Name
	owner.Name = "ScratchDto"
	if a.validateInternalModels(ds) == nil {
		t.Error("changed component owner name authorized")
	}
	owner.Name = ownerName
	ownerCopy := *owner
	a.components["Holder"] = &ownerCopy
	if a.validateInternalModels(ds) == nil {
		t.Error("substituted component owner authorized")
	}
	a.components["Holder"] = owner
	if origin.Helper == nil || !origin.Helper.AbsentSerializer || len(origin.Fields) != 2 || len(model.Nested) != 1 {
		t.Fatal("missing sharedactualzerodomaincodec")
	}
	model.ComponentDefault = "{}"
	if a.validateInternalModels(ds) == nil {
		t.Error("unsourcedliteral-ownerfactory authorized")
	}
	model.ComponentDefault = ""
	savedNullable := model.Nullable
	model.Nullable = !savedNullable
	if a.validateInternalModels(ds) == nil {
		t.Error("changed ownernullableintent authorized")
	}
	model.Nullable = savedNullable
	savedFlag := origin.Helper.AbsentSerializer
	origin.Helper.AbsentSerializer = false
	if a.validateInternalModels(ds) == nil {
		t.Error("removed strictabsencecodec authorized")
	}
	origin.Helper.AbsentSerializer = savedFlag
	savedName := origin.Helper.ClassName
	origin.Helper.ClassName = "JsonElement"
	if a.validateInternalModels(ds) == nil {
		t.Error("renamed actualnestedcodec authorized")
	}
	origin.Helper.ClassName = savedName
	for _, mutate := range []func(*Property){func(p *Property) { p.OmitDefault = false }, func(p *Property) { p.DefaultNull = false }, func(p *Property) { p.Type = TypeRef{Kind: JSON} }, func(p *Property) { p.Type.Serializer = "" }, func(p *Property) { p.Default = "null" }} {
		saved := model.Properties[0]
		mutate(&model.Properties[0])
		if a.validateInternalModels(ds) == nil {
			t.Error("changed actualliteralfalsegetter authorized")
		}
		model.Properties[0] = saved
	}
	originalNested := model.Nested
	copiedHelper := *origin.Helper
	model.Nested = []Declaration{&copiedHelper}
	if a.validateInternalModels(ds) == nil {
		t.Error("copied nested codec authorized")
	}
	model.Nested = originalNested
	savedProperties := model.Properties
	model.Properties = append([]Property(nil), savedProperties...)
	if a.validateInternalModels(ds) == nil {
		t.Error("copied fieldstorage authorized")
	}
	model.Properties = savedProperties
	delete(a.literalFalseFieldOrigins, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missinglastliteralfieldproof authorized")
	}
	a.literalFalseFieldOrigins[model] = origin
	without := []Declaration{}
	for _, d := range ds {
		if d != model {
			without = append(without, d)
		}
	}
	if a.validateInternalModels(without) == nil {
		t.Error("missing actualowner authorized")
	}
	copiedModel := *model
	if a.validateInternalModels(append(without, &copiedModel)) == nil {
		t.Error("copied actualowner authorized")
	}
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	props := get(a.modelOrigins[model], "properties")
	copyProps := get(get(get(get(copied, "components"), "schemas"), "Holder"), "properties")
	for i, m := range props.Members() {
		if m.Name == "optional" {
			props.Members()[i].Value = get(copyProps, "optional")
			if a.validateInternalModels(ds) == nil {
				t.Error("sameboolreplacement authorized")
			}
			props.Members()[i] = m
		}
	}
	oldElement := model.Properties[0].Type.Element
	model.Properties[0].Type.Element = &model.Properties[0].Type
	if a.validateInternalModels(ds) == nil {
		t.Error("cyclicfieldtype authorized")
	}
	model.Properties[0].Type.Element = oldElement
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
