package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestNestedStringEnumDefaultActualVocabularySourceAndTypeAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"rows":{"type":"array","items":{"additionalProperties":{"enum":[null,"one","two"],"nullable":true}},"default":[{"z":null,"a":"one"}]}}},"Use":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/rows/items/additionalProperties"}}}},"Bridge":{"$ref":"#/components/schemas/Use/properties/values"},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows/items/additionalProperties"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.nestedenums", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	if failure := a.validateInternalModels(ds); failure != nil {
		t.Fatal(failure)
	}
	var owner *Model
	for _, d := range ds {
		if d.Name() == "OwnerDto" {
			owner = d.(*Model)
		}
	}
	if owner == nil || len(owner.Nested) != 1 || len(a.borrowedEnumLeaves) != 1 {
		t.Fatal("actual source not retained")
	}
	actual := owner.Nested[0].(*StringEnum)
	saved := owner.Nested
	for _, bad := range [][]Declaration{nil, {&StringEnum{ClassName: actual.ClassName, Members: actual.Members}}, {actual, actual}} {
		owner.Nested = bad
		if a.validateInternalModels(ds) == nil {
			t.Fatal("missing/copied/duplicate actual accepted")
		}
	}
	owner.Nested = saved
	origin := a.enumOrigins[actual]
	changed := origin
	changed.Nullable = !changed.Nullable
	a.enumOrigins[actual] = changed
	if a.validateInternalModels(ds) == nil {
		t.Fatal("nullable origin fabricated")
	}
	a.enumOrigins[actual] = origin
	field := get(get(a.components["Owner"].Schema, "properties"), "rows")
	leaf := get(get(field, "items"), "additionalProperties")
	values := get(leaf, "enum")
	member := values.Items()[0]
	copyTree, e := document.Load([]byte(`{"value":null}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	values.Items()[0] = get(copyTree, "value")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte null member accepted")
	}
	values.Items()[0] = member
	old := actual.Members[0]
	actual.Members[0].Wire = "future"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("altered lexical vocabulary accepted")
	}
	actual.Members[0] = old
	originalSchema := a.components["Owner"].Schema
	copied, e := document.Load([]byte(`{"properties":{"rows":{"type":"array","items":{"additionalProperties":{"enum":[null,"one","two"],"nullable":true}},"default":[{"z":null,"a":"one"}]}}}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a.components["Owner"].Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte copied catalog accepted")
	}
	a.components["Owner"].Schema = originalSchema
	if failure := a.validateInternalModels(ds); failure != nil {
		t.Fatal(failure)
	}
	ty := owner.Properties[0].Type.WithoutRootNull()
	if ty.Kind != List || ty.Element.Kind != Map || ty.Element.Element.Kind != Nullable || ty.Element.Element.Element.Kind != Named || ty.Element.Element.Element.Name != actual.ClassName || ty.Element.Element.Element.Serializer != "" {
		t.Fatal("native/customcodec/text authority fabricated", ty)
	}
	first := nameLockShapeV1(owner, "org.example.nestedenums")
	defaults := owner.Properties[0].Default
	owner.Properties[0].Default = ""
	if first == nameLockShapeV1(owner, "org.example.nestedenums") {
		t.Fatal("effective default lost")
	}
	owner.Properties[0].Default = defaults
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
