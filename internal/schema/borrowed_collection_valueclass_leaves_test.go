package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionValueClassLeavesActualPhysicalLayoutAndChecks(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Count":{"type":["integer","null"],"format":"int64","x-kotlin-value-class":true,"minimum":2},"Alias":{"$ref":"#/components/schemas/Count"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedwrappers", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	c := a.components["Count"]
	if len(a.borrowedValueClassLeaves) != 1 || a.borrowedValueClassLeaves[c].Schema != c.Schema {
		t.Fatal("original wrapper permission missing")
	}
	alias := a.componentType(a.components["Bridge"])
	item := bare(alias).Element
	if item == nil || item.Kind != InlineValue || item.Name != "org.example.borrowedwrappers.models.CountDto" || item.Element == nil || item.Element.Name != "kotlin.Long" || item.Slots() != 2 || alias.Slots() != 1 {
		t.Fatal("borrowed wrapper lost physical scalar layout", alias)
	}
	var actual *PrimitiveValueClass
	var without []Declaration
	for _, d := range ds {
		if d.Name() == "CountDto" {
			actual = d.(*PrimitiveValueClass)
		} else {
			without = append(without, d)
		}
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &PrimitiveValueClass{ClassName: actual.ClassName, Scalar: actual.Scalar, Validation: actual.Validation}), append(append([]Declaration{}, ds...), actual)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated original physical wrapper authorized")
		}
	}
	scalar := actual.Scalar
	actual.Scalar = "kotlin.Double"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("mutated physical scalar authorized")
	}
	actual.Scalar = scalar
	checks := *actual.Validation.Minimum
	actual.Validation.Minimum.Literal = "3L"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("mutated wrapper checks authorized")
	}
	*actual.Validation.Minimum = checks
	origin := a.valueClassOrigins[actual]
	badOrigin := origin
	badOrigin.Nullable = !origin.Nullable
	a.valueClassOrigins[actual] = badOrigin
	if a.validateInternalModels(ds) == nil {
		t.Fatal("mutated wrapper nullable origin authorized")
	}
	a.valueClassOrigins[actual] = origin
	originalSource := c.Schema
	copied, copyErr := document.Load([]byte(`{"type":["integer","null"],"format":"int64","x-kotlin-value-class":true,"minimum":2}`), document.JSON, "")
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	c.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte copied physical wrapper source authorized")
	}
	c.Schema = originalSource
	aliasComponent := a.components["Alias"]
	height := aliasComponent.AliasHeight
	aliasComponent.AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("mutated physical wrapper alias suffix cache authorized")
	}
	aliasComponent.AliasHeight = height
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	use := a.borrowedValueClassLeaves[c]
	members := c.Schema.Members()
	index := -1
	for i, member := range members {
		if member.Name == "minimum" {
			index = i
		}
	}
	originalMember := members[index]
	copyTree, loadErr := document.Load([]byte(`{"value":2}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	members[index].Value = get(copyTree, "value")
	// Updating the ordered snapshot cannot authorize a different keyed source node.
	use.SchemaMembers[index] = members[index]
	if a.validateInternalModels(ds) == nil {
		t.Fatal("ordered-keyed wrapper source divergence authorized")
	}
	members[index] = originalMember
	use.SchemaMembers[index] = originalMember
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("physical wrapper binding mutated source tree")
	}
}
