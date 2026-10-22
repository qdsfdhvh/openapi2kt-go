package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestNamedNativeEnumBorrowingNeedsActualVocabularyAndScalarExport(t *testing.T) {
	testNamedNativeEnumSourceAuthority(t, false)
}

func TestNativeEnumUniqueRetainsActualVocabularyAndScalarExport(t *testing.T) {
	testNamedNativeEnumSourceAuthority(t, true)
}

func testNamedNativeEnumSourceAuthority(t *testing.T, unique bool) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2,3,4],"minimum":2},"Alias":{"$ref":"#/components/schemas/Code"},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Rows":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	if unique {
		input = []byte(strings.Replace(string(input), `"type":"array","items"`, `"type":"array","uniqueItems":true,"items"`, 1))
	}
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.enums", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	c := a.components["Code"]
	origin := a.nativeEnumOrigins[c]
	if origin == nil || origin.Validation != c.Validation || origin.Vocabulary != get(c.Schema, "enum") || origin.Actual == nil || a.components["Bridge"].InternalModelTarget.CanonicalNativeEnum != c {
		t.Fatal(origin)
	}
	if err := a.validateNativeEnumBindings(ds); err != nil {
		t.Fatal(err)
	}
	without := []Declaration{}
	for _, d := range ds {
		if d != origin.Actual {
			without = append(without, d)
		}
	}
	if err := a.validateNativeEnumBindings(without); err == nil {
		t.Fatal("missing original export allowed")
	}
	if err := a.validateNativeEnumBindings(append(append([]Declaration{}, ds...), origin.Actual)); err == nil {
		t.Fatal("repeated original export allowed")
	}
	replacement := &ScalarAlias{ClassName: origin.Actual.Name(), Scalar: origin.Scalar}
	if err := a.validateNativeEnumBindings(append(without, replacement)); err == nil {
		t.Fatal("same-name replacement export allowed")
	}
	copy, e := document.Load([]byte(`{"value":1}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	old := origin.Vocabulary.Items()[0]
	origin.Vocabulary.Items()[0] = get(copy, "value")
	if err := a.validateNativeEnumBindings(ds); err == nil {
		t.Fatal("same-byte vocabulary member replacement allowed")
	}
	origin.Vocabulary.Items()[0] = old
	oldCheck := c.Validation.Enum[0]
	c.Validation.Enum[0] = "9"
	if err := a.validateNativeEnumBindings(ds); err == nil {
		t.Fatal("cached membership mutation allowed")
	}
	c.Validation.Enum[0] = oldCheck
	if err := a.validateNativeEnumBindings(ds); err != nil {
		t.Fatal(err)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != string(input) {
		t.Fatal("source rewritten")
	}
}
