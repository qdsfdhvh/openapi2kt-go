package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestReferencedStringEnumNullMemberDefaultsOriginalSourceAndNarrowing(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":["string","null"],"enum":[null,"on"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Status","nullable":false}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":["on"]}}}}}}`)
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
	original := a.components["Status"].Schema
	if err != nil || node.element.schema != original || !node.element.stringEnum || node.element.ty.Kind != Named || node.element.ty.Name != "org.example.defaults.models.StatusDto" {
		t.Fatal("source/classifier/narrowed view", node, err)
	}
	// Source null does not override the actual nearest false typed view.
	valueTree, err := document.Load([]byte(`{"value":null}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	value := get(valueTree, "value")
	if _, err := node.element.value(value, "/default/0"); err == nil || err.Code != "E_SCHEMA_INVALID" {
		t.Fatal("vocabulary null granted nullable view", err)
	}
	// Default planning reads the original vocabulary, including its null node.
	members := get(original, "enum").Items()
	if members[0].Kind() != document.Null || get(node.element.schema, "enum").Items()[0] != members[0] {
		t.Fatal("original vocabulary null node lost")
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source rewritten")
	}
}
