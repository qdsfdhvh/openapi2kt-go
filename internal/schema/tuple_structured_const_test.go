package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestStructuredLiteralCaptureDoesNotResolveDataKeys(t *testing.T) {
	root := loadComponentDocument(t, `{"Only":{"const":{"allOf":[{"$ref":"#/data-only"}],"properties":{"description":{"default":{"allOf":[{"$ref":"#/also-data"}]}}},"description":"data"}}}`)
	value := get(get(get(root, "components"), "schemas"), "Only")
	a := analyzer{root: root, components: map[string]*component{}}
	nodes, refs, err := a.borrowedAllOfSourceBudget(value, "/components/schemas/Only", "/components/schemas/Only", nil)
	if err != nil || len(refs) != 0 {
		t.Fatal("literal keys acquired shape/ref authority", err, len(refs))
	}
	data := get(get(value, "const"), "description")
	found := false
	for _, n := range nodes {
		if n.Node == data {
			found = true
		}
	}
	if !found {
		t.Fatal("metadata-named literal data disappeared from original snapshots")
	}
}

func TestTupleStructuredConstOriginalCodecAndDataProof(t *testing.T) {
	root := loadComponentDocument(t, `{"Only":{"const":{"fixed":1}},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"const":{"description":{"allOf":[{"$ref":"#/data-only"}]}}}],"default":[{"description":{"allOf":[{"$ref":"#/data-only"}]}}]}}}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.structuredtuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
		t.Fatal("copied structured element acquired original source proof")
	}
	outer.Element = leaf
	name := leaf.Name
	leaf.Name = "kotlinx.serialization.json.JsonArray"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed JSON kind codec retained tuple authority")
	}
	leaf.Name = name
	var declaration *ScalarAlias
	for _, d := range ds {
		if d.Name() == "OnlyDto" {
			declaration = d.(*ScalarAlias)
		}
	}
	declaration.Scalar = "kotlin.String"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed actual structured root export was accepted")
	}
	declaration.Scalar = "kotlinx.serialization.json.JsonObject"
	keys := a.tupleLiterals.StructuredKeys
	a.tupleLiterals.StructuredKeys = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("missing canonical original literal proof was accepted")
	}
	a.tupleLiterals.StructuredKeys = keys
	member := get(get(get(a.components["Holder"].Schema, "properties"), "rows"), "prefixItems").Items()[0]
	data := get(member, "const")
	original := data.Members()[0].Value
	encoded, _ := original.MarshalJSON()
	replacement, fail := document.Load(encoded, document.JSON, "")
	if fail != nil {
		t.Fatal(fail)
	}
	data.Members()[0].Value = replacement
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte metadata-named JSON data copy acquired original authority")
	}
	data.Members()[0].Value = original
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
}
