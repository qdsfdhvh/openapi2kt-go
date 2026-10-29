package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleRecursiveCoverActualGraphAndTypeAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":`+jsonGraphCover("Other", "Other")+`,"Other":`+jsonGraphCover("Value", "Value")+`,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Maybe"}]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.cover", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	var rows, value *ValueAlias
	for _, d := range ds {
		if d.Name() == "RowsDto" {
			rows = d.(*ValueAlias)
		} else if d.Name() == "ValueDto" {
			value = d.(*ValueAlias)
		}
	}
	if rows == nil || value == nil || len(a.tupleCovers.Order) != 1 {
		t.Fatal("missing actual tuple cover declarations")
	}
	copyAlias := *value
	duplicate := append(append([]Declaration{}, ds...), &copyAlias)
	if a.validateTuplePlan(&Plan{Models: duplicate}) == nil {
		t.Fatal("duplicate same-name actual export retained tuple authority")
	}
	element := rows.Type.Element
	copyElement := *element
	rows.Type.Element = &copyElement
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte copied consuming element acquired tuple authority")
	}
	rows.Type.Element = element
	old := value.Type
	value.Type = TypeRef{Kind: Scalar, Name: "kotlinx.serialization.json.JsonElement"}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-spelling scalar export acquired complete graph authority")
	}
	value.Type = old
	terminal := a.components["Value"].Schema
	height := a.recursiveJSONHeights[terminal]
	a.recursiveJSONHeights[terminal]++
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated cached SCC height retained tuple authority")
	}
	a.recursiveJSONHeights[terminal] = height
	source := a.tupleCovers.Order[0]
	source.Steps++
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated erased source height retained tuple authority")
	}
	source.Steps--
	name := source.Root.Name
	source.Root.Name = "DifferentDto"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated actual canonical name retained tuple authority")
	}
	source.Root.Name = name
	other := get(get(a.components["Other"].Schema, "oneOf").Items()[4], "items")
	data, _ := other.MarshalJSON()
	copyRoot, failure := document.Load(append(append([]byte(`{"value":`), data...), '}'), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	branch := get(a.components["Other"].Schema, "oneOf").Items()[4]
	for i, member := range branch.Members() {
		if member.Name != "items" {
			continue
		}
		branch.Members()[i].Value = get(copyRoot, "value")
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("same-byte reachable graph source acquired authority")
		}
		branch.Members()[i].Value = other
	}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple cover planning mutated original graph")
	}
}
