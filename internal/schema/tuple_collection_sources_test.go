package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleCollectionRetainedSourceAndBacking(t *testing.T) {
	root := loadComponentDocument(t, `{"Numbers":{"type":"array","items":{"type":"integer"}},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Numbers"}],"default":[[1]]}}}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{referencedDefaults: &pending, root: root, options: Options{PackageName: "org.example.collectiontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	outer := holder.Properties[0].Type.Element
	leaf := outer.Element
	copyType := *leaf
	outer.Element = &copyType
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied collection element acquired source authority")
	}
	outer.Element = leaf
	name := leaf.Name
	leaf.Name = "org.example.collectiontuples.models.OtherDto"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("replaced collection classifier kept original binding")
	}
	leaf.Name = name
	backing := a.components["Numbers"].AliasType
	a.components["Numbers"].AliasType = TypeRef{Kind: List, Element: &TypeRef{Kind: Scalar, Name: "kotlin.String"}}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed original backing kept tuple proof")
	}
	a.components["Numbers"].AliasType = backing
	proof := a.tupleObjects.Proof.Collections
	a.tupleObjects.Proof.Collections = nil
	if a.validateInternalModels(ds) == nil || a.validateTuplePlan(plan) == nil {
		t.Fatal("missing collection source proof was accepted")
	}
	a.tupleObjects.Proof.Collections = proof
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	member := get(get(get(a.components["Holder"].Schema, "properties"), "rows"), "prefixItems")
	original := member.Items()[0]
	bytes, _ := original.MarshalJSON()
	copyNode, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	member.Items()[0] = copyNode
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte source reference copy acquired permission")
	}
	member.Items()[0] = original
}

func TestTupleCollectionDiscardedValidationIsolation(t *testing.T) {
	root := loadComponentDocument(t, `{"Choice":{"enum":["ready","paused"]},"Rows":{"type":"array","prefixItems":[{"type":"array","items":{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"}}}}],"items":true}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.collectiontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 || len(a.modelOrigins) != 0 || len(pending) != 0 || len(a.components) != 2 {
		t.Fatal("discarded collection validation leaked registrations", len(ds), len(a.modelOrigins), len(pending), len(a.components))
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	plan, warnings, failure := analyzeDocument(root, a.options, true, nil)
	if plan != nil || warnings != nil || failure != nil {
		t.Fatal("collection validation returned emission candidate", failure)
	}
}

func TestTupleCollectionReferencedLeafSourceProof(t *testing.T) {
	root := loadComponentDocument(t, `{"Base":{"properties":{"name":{"type":"string"}}},"Rows":{"type":"array","prefixItems":[{"type":"array","items":{"$ref":"#/components/schemas/Base"}}],"items":true}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.collectiontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	base := a.components["Base"]
	pointer := base.Pointer
	base.Pointer = "/components/schemas/Other"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed discarded collection leaf catalog retained source proof")
	}
	base.Pointer = pointer
	props := get(base.Schema, "properties")
	original := props.Members()[0].Value
	bytes, _ := original.MarshalJSON()
	copyNode, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	props.Members()[0].Value = copyNode
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte referenced leaf child acquired original tuple authority")
	}
	props.Members()[0].Value = original
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
