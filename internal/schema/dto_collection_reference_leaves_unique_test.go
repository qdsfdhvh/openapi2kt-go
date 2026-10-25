package schema

import (
	"testing"
)

func dtoCollectionReferenceLeavesAnalysis(t *testing.T) (*analyzer, *Plan) {
	t.Helper()
	root := loadComponentDocument(t, `{"Scalar":{"type":"integer","minimum":0},"Item":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Scalar"}}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.leafrefs", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal(e)
	}
	return a, &Plan{Models: ds}
}

func TestDTOCollectionReferenceLeavesOriginalCheckValues(t *testing.T) {
	a, plan := dtoCollectionReferenceLeavesAnalysis(t)
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
	checks := a.components["Scalar"].Validation
	if checks == nil || checks.Minimum == nil {
		t.Fatal("missing original scalar checks")
	}
	checks.Minimum.Literal = "2"
	if e := a.validateDTOUniqueUses(plan); e == nil {
		t.Fatal("changed shared actual/original check value authorized")
	}
}

func TestDTOCollectionReferenceLeavesFirstBarrierCheckIdentity(t *testing.T) {
	a, plan := dtoCollectionReferenceLeavesAnalysis(t)
	item := componentDeclaration(plan.Models, "ItemDto").(*Model)
	leaf := item.Properties[0].Type.WithoutRootNull().Element
	if leaf == nil || leaf.Validation == nil {
		t.Fatal("missing actual leaf checks")
	}
	copy := *leaf.Validation
	leaf.Validation = &copy
	if e := a.validateDTOUniqueUses(plan); e == nil {
		t.Fatal("same-byte copied actual check object before first barrier authorized")
	}
}

func TestDTOCollectionReferenceLeavesOriginalCheckRestoration(t *testing.T) {
	a, plan := dtoCollectionReferenceLeavesAnalysis(t)
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
	leaf := componentDeclaration(plan.Models, "ItemDto").(*Model).Properties[0].Type.WithoutRootNull().Element
	original := leaf.Validation
	copy := *original
	leaf.Validation = &copy
	if e := a.validateDTOUniqueUses(plan); e == nil {
		t.Fatal("copied actual check allowed")
	}
	leaf.Validation = original
	value := original.Minimum.Literal
	original.Minimum.Literal = "2"
	if e := a.validateDTOUniqueUses(plan); e == nil {
		t.Fatal("changed bound value allowed")
	}
	original.Minimum.Literal = value
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original check rejected", e)
	}
}
