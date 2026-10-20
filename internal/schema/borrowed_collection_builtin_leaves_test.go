package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionBuiltinLeavesOriginalScalarChecksAndDeclaration(t *testing.T) {
	for _, format := range []string{"", "uuid"} {
		t.Run(format, func(t *testing.T) { testBorrowedBuiltinOriginalScalar(t, format) })
	}
}

func testBorrowedBuiltinOriginalScalar(t *testing.T, format string) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Text":{"type":["string","null"],"minLength":2},"Maybe":{"$ref":"#/components/schemas/Text"},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/False"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	if format != "" {
		input = []byte(strings.Replace(string(input), `"type":["string","null"]`, `"type":["string","null"],"format":"`+format+`"`, 1))
	}
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.builtinleaves", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.borrowedBuiltinLeaves) != 1 {
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
	old := *c.Validation.MinLength
	*c.Validation.MinLength = 3
	if a.validateInternalModels(ds) == nil {
		t.Fatal("in-place original scalar check changed but authorized")
	}
	*c.Validation.MinLength = old
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
