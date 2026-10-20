package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedBytesDefaultsUseOriginalBackingAndSource(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bytes":{"type":"string","format":"byte","nullable":true},"Alias":{"$ref":"#/components/schemas/Bytes"},"Hops":{"properties":{"v":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/v","default":[-128,127]}}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.bytes", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
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
		t.Fatal(pending)
	}
	plan := pending[0]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	c := a.components["Bytes"]
	if err != nil || node.schema != c.Schema || node.pointer != c.Pointer || node.ty.Kind != Scalar || node.ty.Name != "kotlin.ByteArray" || node.ty.Serializer != "" || node.ty.Validation != nil || node.ty.Slots() != 1 {
		t.Fatal(node, err)
	}
	if plan.Model.Properties[plan.Index].Default != "`openapi-kt default tree Scalar ByteArray`(\"[-128,127]\")" {
		t.Fatal(plan.Model.Properties[plan.Index])
	}
	// Final binding still demands the original retained declaration, not its name.
	missing := []Declaration{}
	var original Declaration
	for _, d := range ds {
		if d.Name() == c.Name {
			original = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateBytesBindings(missing); err == nil {
		t.Fatal("missing backing authorized")
	}
	replacement := &ValueAlias{ClassName: c.Name, Type: a.componentType(c)}
	if err := a.validateBytesBindings(append(append([]Declaration{}, missing...), replacement)); err == nil {
		t.Fatal("replacement backing authorized")
	}
	if err := a.validateBytesBindings(append(append([]Declaration{}, ds...), original)); err == nil {
		t.Fatal("repeated backing authorized")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
