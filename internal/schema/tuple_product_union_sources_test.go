package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

const tupleProductSource = `{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"name"},"name":{"type":"string"}},"required":["name"]}]},{"properties":{"common":{"type":"string"}}}]}`

func TestTupleProductInternalReference(t *testing.T) {
	root := loadComponentDocument(t, `{"Owner":{"properties":{"choice":`+tupleProductSource+`}},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Owner/properties/choice"}],"default":[{"kind":"name","name":"ready"}]}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.producttuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatalf("%v at %s tasks=%d unions=%d borrowed=%d proof=%v", err, *err.Pointer, len(a.tupleObjects.Tasks), len(a.tupleObjects.Unions), len(a.borrowedUnionLeaves), a.tupleObjects.Proof != nil)
	}
	if err := a.validateTuplePlan(&Plan{Models: ds}); err != nil {
		t.Fatal(err)
	}
}

func TestTupleProductInternalIsolatedSourceCompletion(t *testing.T) {
	root := loadComponentDocument(t, `{"Owner":{"properties":{"choice":`+tupleProductSource+`}},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Owner/properties/choice"}],"default":[{"kind":"name","name":"ready"}]}}`)
	a := analyzer{tupleSourceValidation: true, root: root, options: Options{PackageName: "org.example.producttuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatalf("analyze %v at %s", err, *err.Pointer)
	}
	ds, err = a.attachTupleSourceModels(ds, nil)
	if err != nil {
		t.Fatalf("attach %v at %s", err, *err.Pointer)
	}
	if err := a.finalizeTupleCollectionUses(); err != nil {
		t.Fatalf("finalize %v at %s", err, *err.Pointer)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatalf("internal %v at %s", err, *err.Pointer)
	}
	if err := a.validateTuplePlan(&Plan{Models: ds}); err != nil {
		t.Fatalf("tuple %v at %s", err, *err.Pointer)
	}
}

func TestTupleProductRetainedSelectorAndOriginalGraph(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[`+tupleProductSource+`]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.producttuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	var union *ObjectUnion
	for _, d := range ds {
		if d, ok := d.(*ObjectUnion); ok {
			union = d
		}
	}
	if union == nil || len(union.TagCombinations) == 0 {
		t.Fatal("missing actual Cartesian selector")
	}
	value := union.TagCombinations[0].Matches[0].Values[0]
	union.TagCombinations[0].Matches[0].Values[0] = "other"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed conjunction vocabulary kept tuple authority")
	}
	union.TagCombinations[0].Matches[0].Values[0] = value
	variant := union.Variants[0]
	name := variant.ClassName
	variant.ClassName = "OtherVariant"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("renamed actual product variant kept original tuple codec authority")
	}
	variant.ClassName = name
	copyVariant := *variant
	union.Variants[0] = &copyVariant
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied product constructor acquired source authority")
	}
	union.Variants[0] = variant
	origin := a.modelOrigins[variant]
	a.modelOrigins[variant] = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("actual product variant lost original model source without losing permission")
	}
	a.modelOrigins[variant] = origin
	for source, fact := range a.tupleObjects.Proof.Unions {
		changed := fact
		changed.Product = false
		a.tupleObjects.Proof.Unions[source] = changed
		if a.validateInternalModels(ds) == nil {
			t.Fatal("product source masqueraded as ordinary union fact")
		}
		a.tupleObjects.Proof.Unions[source] = fact
	}
	member := get(a.components["Rows"].Schema, "prefixItems").Items()[0]
	allof := get(member, "allOf")
	original := allof.Items()[0]
	bytes, _ := original.MarshalJSON()
	replacement, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	allof.Items()[0] = replacement
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original product operand replacement kept source proof")
	}
	allof.Items()[0] = original
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("product tuple planning rewrote original document")
	}
}
