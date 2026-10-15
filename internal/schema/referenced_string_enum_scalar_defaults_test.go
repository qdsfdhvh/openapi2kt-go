package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestReferencedStringEnumScalarDefaultsSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"code":{"type":"string","enum":["on","off"]}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/code"},"Alias":{"$ref":"#/components/schemas/Bridge"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false,"default":"on"}}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	plan := pending[0]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	source := get(get(a.components["Source"].Schema, "properties"), "code")
	actual := plan.Model.Properties[plan.Index].Type.WithoutRootNull()
	if err != nil || node.schema != source || node.pointer != "/components/schemas/Source/properties/code" || !node.stringEnum || node.native != nil || node.ty.Kind != Named || actual.Name != "org.example.defaults.models.SourceDto.Code" || node.ty.deferredNominal == nil || node.ty.deferredNominal.Schema != source {
		t.Fatal("original source/actual classifier", node, actual, err)
	}
	if plan.Model.Properties[plan.Index].Default != "`openapi-kt default enum scalar`(\"\\\"on\\\"\")" {
		t.Fatal("unbound validated JSON marker")
	}
	if _, err := a.referencedDefaultSchemaRole(node.schema, node.pointer, node.ty, 0, false, noReferencedDefaultRole); err == nil {
		t.Fatal("Named spelling granted source role")
	}
	if _, err := a.referencedDefaultSchemaRole(node.schema, node.pointer, TypeRef{Kind: Scalar, Name: "kotlin.String"}, 0, false, referencedStringEnumOwnDefault); err == nil {
		t.Fatal("plain scalar swallowed enum classifier")
	}
	// An annotation does not replace the existing actual-owner authority barrier.
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() != "SourceDto" {
			without = append(without, d)
		}
	}
	if err := a.validateInternalModels(without); err == nil {
		t.Fatal("missing actual lexical owner authorized")
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source tree mutated")
	}
}
