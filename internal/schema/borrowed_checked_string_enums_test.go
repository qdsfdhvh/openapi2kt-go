package schema

import "testing"

func TestBorrowedCheckedStringEnumOriginalChecksAndDeclaration(t *testing.T) {
	root := loadComponentDocument(t, `{"ZOwner":{"properties":{"word":{"enum":["one","two","x"],"minLength":2,"maxLength":6},"tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}}}}},"Word":{"$ref":"#/components/schemas/ZOwner/properties/word"},"Tree":{"$ref":"#/components/schemas/ZOwner/properties/tree"},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Word"},"tree":{"$ref":"#/components/schemas/Tree"}}}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.checked", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	owner := componentModel(ds, "ZOwnerDto")
	use := componentModel(ds, "UseDto")
	if owner == nil || use == nil || len(a.borrowedEnumLeaves) != 2 {
		t.Fatal("missing original checked declaration sources")
	}
	word := get(get(a.components["ZOwner"].Schema, "properties"), "word")
	checked := a.checkedEnumSources[word]
	if checked == nil || checked.Checks == nil || owner.Properties[0].Type.WithoutRootNull().Validation != checked.Checks || use.Properties[0].Type.WithoutRootNull().Validation != checked.Checks || defaultBacking(a.componentType(a.components["Word"])).WithoutRootNull().Validation != checked.Checks {
		t.Fatal("forward source/alias/consumer did not retain original checks")
	}
	proof := a.borrowedEnumLeaves[word]
	if proof == nil || proof.Checked != checked || !checked.Declarations[proof.Actual] {
		t.Fatal("checked borrowing has no actual retained source marker")
	}
	delete(checked.Declarations, proof.Actual)
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("unmarked retained enum acquired checked-source permission")
	}
	checked.Declarations[proof.Actual] = true
	minimum := *checked.Checks.MinLength
	*checked.Checks.MinLength++
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("mutated original check values retained source permission")
	}
	*checked.Checks.MinLength = minimum
	actual := proof.Actual
	original := actual.Members[0]
	actual.Members[0].Wire = "different"
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("mutated actual checked vocabulary retained permission")
	}
	actual.Members[0] = original
	getter := &owner.Properties[0].Type
	for getter.Kind == Nullable {
		getter = getter.Element
	}
	getterCheck := getter.Validation
	getter.Validation = nil
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("checked source getter lost its enforcing check without rejection")
	}
	getter.Validation = getterCheck
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
