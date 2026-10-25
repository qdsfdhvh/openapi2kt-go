package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseInlineCollectionDefaultActualAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Denied":false,"Holder":{"required":["required"],"properties":{"blocked":{"type":"array","items":{"$ref":"#/components/schemas/Denied"},"default":[]},"required":{"additionalProperties":{"$ref":"#/components/schemas/Denied"},"default":{}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsevalues", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
		t.Fatal("missing actual model")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Property)
	}{
		{"initializer", func(p *Property) { p.Default = "" }},
		{"omission", func(p *Property) { p.OmitDefault = true }},
		{"presence", func(p *Property) { p.DefaultNull = false }},
		{"serializer", func(p *Property) { p.Type.Serializer = "invalid" }},
		{"type", func(p *Property) { p.Type = TypeRef{Kind: JSON} }},
		{"wire", func(p *Property) { p.WireName = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saved := model.Properties[0]
			tc.mutate(&model.Properties[0])
			defer func() { model.Properties[0] = saved }()
			if a.validateInternalModels(ds) == nil {
				t.Fatal("changed actual false property authorized")
			}
		})
	}
	savedInitializer := model.Properties[1].Default
	savedRequiredNull := model.Properties[1].DefaultNull
	model.Properties[1].Default = "emptyMap()"
	if a.validateInternalModels(ds) == nil {
		t.Error("required field acquired initializer")
	}
	model.Properties[1].Default = savedInitializer
	model.Properties[1].DefaultNull = true
	if a.validateInternalModels(ds) == nil {
		t.Error("required own-default acquired incidental missing state")
	}
	model.Properties[1].DefaultNull = savedRequiredNull
	proof := a.falseInlineCollectionOrigins[model]
	delete(a.falseInlineCollectionOrigins, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing last actual field collection proof authorized")
	}
	a.falseInlineCollectionOrigins[model] = proof
	saved := model.Properties
	model.Properties = append([]Property{}, saved...)
	if a.validateInternalModels(ds) == nil {
		t.Error("copied same-byte field storage authorized")
	}
	model.Properties = saved
	without := []Declaration{}
	for _, d := range ds {
		if d != model {
			without = append(without, d)
		}
	}
	if a.validateInternalModels(without) == nil {
		t.Error("missing actual field model authorized")
	}
	copyModel := *model
	if a.validateInternalModels(append(without, &copyModel)) == nil {
		t.Error("same-byte copied model authorized")
	}
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	owner := a.components["Holder"].Schema
	copied := get(get(get(copyRoot, "components"), "schemas"), "Holder")
	for i, m := range owner.Members() {
		if m.Name == "required" {
			owner.Members()[i].Value = get(copied, "required")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte required member replacement authorized")
			}
			owner.Members()[i] = m
		}
	}
	// A copied leaf must not inherit the original retained codec proof.
	savedElement := model.Properties[0].Type.Element
	model.Properties[0].Type.Element = &model.Properties[0].Type
	if a.validateInternalModels(ds) == nil {
		t.Error("cyclic false field type admitted")
	}
	model.Properties[0].Type.Element = savedElement
	clonedElement := *savedElement
	model.Properties[0].Type.Element = &clonedElement
	if a.validateInternalModels(ds) == nil {
		t.Error("same-byte copied false type tree authorized")
	}
	model.Properties[0].Type.Element = savedElement
	fieldSource := get(get(get(owner, "properties"), "blocked"), "items")
	copiedField := get(get(get(copied, "properties"), "blocked"), "items")
	for i, m := range fieldSource.Members() {
		if m.Name == "$ref" {
			fieldSource.Members()[i].Value = get(copiedField, "$ref")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte field reference member replacement authorized")
			}
			fieldSource.Members()[i] = m
		}
	}
	for _, d := range ds {
		if root, ok := d.(*FalseValue); ok {
			saved := root.AbsentSerializer
			root.AbsentSerializer = !saved
			if a.validateInternalModels(ds) == nil {
				t.Error("unbound absence codec flag authorized")
			}
			root.AbsentSerializer = saved
		}
	}

	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
