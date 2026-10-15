package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionBigintLeavesOriginalMarkerAndDeclaration(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Text":{"x-native-type":"bigint","nullable":true},"Maybe":{"$ref":"#/components/schemas/Text"},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/False"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.bigintleaves", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.borrowedBigintLeaves) != 1 {
		t.Fatal("named builtin source not retained")
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	c := a.components["Text"]
	var actual Declaration
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() == "TextDto" {
			actual = d
		} else {
			without = append(without, d)
		}
	}
	if actual == nil {
		t.Fatal("physical source export missing")
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &ScalarAlias{ClassName: "TextDto", Scalar: "kotlin.String"}), append(append([]Declaration{}, ds...), actual)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated canonical scalar source authorized")
		}
	}
	scalar := c.Scalar
	c.Scalar = "kotlin.String"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed bigint source mapping authorized")
	}
	c.Scalar = scalar
	original := c.Schema
	bytes, _ := original.MarshalJSON()
	copied, e := document.Load(bytes, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	c.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original named scalar source replaced")
	}
	c.Schema = original
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
