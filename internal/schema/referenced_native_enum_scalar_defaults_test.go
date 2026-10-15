package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestReferencedNativeEnumScalarFieldDefaultsSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bridge":{"$ref":"#/components/schemas/Source/properties/code"},"Alias":{"$ref":"#/components/schemas/Bridge"},"Source":{"properties":{"code":{"type":["integer","null"],"enum":[1,2,3],"minimum":2}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false,"default":2}}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
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
	origin := a.components["Bridge"].InternalModelTarget.BorrowedScalar
	if err != nil || node.schema != origin.Schema || node.ty.Kind != Scalar || node.ty.Name != "kotlin.Int" || node.ty.Validation != origin.Checks || node.native != &a || node.stringEnum || node.pointer != "/components/schemas/Source/properties/code" {
		t.Fatal("source/type/check authority", node, err)
	}
	if plan.Model.Properties[plan.Index].Default != "2" {
		t.Fatal("literal changed")
	}
	if _, err := a.referencedDefaultSchemaRole(node.schema, node.pointer, node.ty, 0, false, noReferencedDefaultRole); err == nil {
		t.Fatal("coincident scalar granted default role")
	}
	if _, err := a.referencedDefaultSchemaRole(node.schema, node.pointer, TypeRef{Kind: Scalar, Name: "kotlin.Long"}, 0, false, referencedScalarOwnDefault); err == nil {
		t.Fatal("wrong scalar mapping accepted")
	}
	// Planning must keep the original borrowed vocabulary/check authority live.
	saved := origin.Checks.Enum[0]
	origin.Checks.Enum[0] = "9"
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("replaced native check authorized")
	}
	origin.Checks.Enum[0] = saved
	if err := a.validateBorrowedRootBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
