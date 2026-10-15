package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionAllOfModelLeavesOriginalShapeRefsAndActualModels(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Base":{"properties":{"description":{"type":"string"}},"required":["description"]},"BaseAlias":{"$ref":"#/components/schemas/Base"},"BaseAgain":{"$ref":"#/components/schemas/BaseAlias"},"Node":{"allOf":[{"$ref":"#/components/schemas/BaseAgain"},{"properties":{"label":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Node"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedallof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var original *Model
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() == "NodeDto" {
			original = d.(*Model)
		} else {
			without = append(without, d)
		}
	}
	if original == nil {
		t.Fatal("canonical allOf constructor absent")
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &Model{ClassName: original.ClassName, Properties: original.Properties}), append(append([]Declaration{}, ds...), original)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated original allOf constructor authorized")
		}
	}
	nullable := original.Nullable
	original.Nullable = !nullable
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed actual allOf nullable view authorized")
	}
	original.Nullable = nullable
	base := a.components["Base"]
	originalBase := base.Schema
	copied, loadErr := document.Load([]byte(`{"properties":{"description":{"type":"string"}},"required":["description"]}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	base.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original allOf shape-ref target replacement authorized")
	}
	base.Schema = originalBase
	shapeAlias := a.components["BaseAlias"]
	originalAliasSchema := shapeAlias.Schema
	copyShapeAlias, aliasErr := document.Load([]byte(`{"$ref":"#/components/schemas/Base"}`), document.JSON, "")
	if aliasErr != nil {
		t.Fatal(aliasErr)
	}
	shapeAlias.Schema = copyShapeAlias
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte intermediate allOf shape alias replacement authorized")
	}
	shapeAlias.Schema = originalAliasSchema
	branches := get(a.components["Node"].Schema, "allOf")
	originalBranch := branches.Items()[0]
	copyBranch, branchErr := document.Load([]byte(`{"$ref":"#/components/schemas/BaseAgain"}`), document.JSON, "")
	if branchErr != nil {
		t.Fatal(branchErr)
	}
	branches.Items()[0] = copyBranch
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original allOf branch replacement authorized")
	}
	branches.Items()[0] = originalBranch
	alias := a.components["Alias"]
	height := alias.AliasHeight
	alias.AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed allOf leaf alias suffix cache authorized")
	}
	alias.AliasHeight = height
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	// A property named "description" is semantic schema, not annotation metadata.
	property := get(get(base.Schema, "properties"), "description")
	for _, use := range a.borrowedAllOfLeaves {
		found := false
		for _, node := range use.Nodes {
			found = found || node.Node == property
		}
		if !found {
			t.Fatal("allOf referenced property schema was omitted from source identity")
		}
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(before) {
		t.Fatal("allOf borrowed binding mutated original input")
	}
}
