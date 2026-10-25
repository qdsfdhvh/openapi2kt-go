package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleEnumOriginalVocabularyAuthorityAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"mode":{"type":"string","enum":["ready","done"],"nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/mode","nullable":false},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Bridge"},{"type":"string"}],"items":{"type":"integer"}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.tupleenums", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	terminal := get(get(a.components["Owner"].Schema, "properties"), "mode")
	member := get(a.components["Probe"].Schema, "prefixItems").Items()[0]
	work := 0
	signature, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work)
	if err != nil || signature.EnumSource != terminal || signature.Nullable || work != 2 || len(a.borrowedEnumLeaves) != 1 {
		t.Fatal("lost original enum/nullable/source work", signature, work, err)
	}
	for _, depth := range []int{MaxSchemaDepth - 2, MaxSchemaDepth - 1} {
		work = 0
		_, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", depth, &work)
		if depth == MaxSchemaDepth-2 && err != nil || depth == MaxSchemaDepth-1 && (err == nil || err.Code != "E_SCHEMA_DEPTH") {
			t.Fatal("enum hop erased source depth", depth, err)
		}
	}
	work = 16383
	if _, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work); err == nil || err.Code != "E_MODEL_LIMIT" {
		t.Fatal("enum source reset work", err)
	}
	without := []Declaration{}
	var owner *Model
	for _, d := range ds {
		if d.Name() == "OwnerDto" {
			owner = d.(*Model)
		} else {
			without = append(without, d)
		}
	}
	if owner == nil || a.validateInternalModels(without) == nil {
		t.Fatal("discarded typed prefix authorized without actual vocabulary owner")
	}
	originalNested := owner.Nested
	actual := originalNested[0].(*StringEnum)
	owner.Nested = append(append([]Declaration{}, originalNested...), actual)
	if a.validateInternalModels(ds) == nil {
		t.Fatal("ambiguous original vocabulary authorized")
	}
	owner.Nested = originalNested
	old := actual.Members[0].Wire
	actual.Members[0].Wire = "changed"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed physical vocabulary authorized")
	}
	actual.Members[0].Wire = old
	saved := a.components["Owner"].Pointer
	a.components["Owner"].Pointer = "/wrong"
	work = 0
	if _, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work); err == nil {
		t.Fatal("copied catalog identity authorized")
	}
	a.components["Owner"].Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple enum mutated original schema")
	}
}
