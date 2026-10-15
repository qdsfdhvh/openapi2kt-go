package schema

import (
	"reflect"
	"testing"

	"openapi-kt/internal/document"
)

func TestNativeEnumNullMemberChecksRetainOriginalVocabularyAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":["integer","null"],"enum":[null,1,2],"minimum":2},"Alias":{"$ref":"#/components/schemas/Code","nullable":false},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.enums", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	c := a.components["Code"]
	origin := a.nativeEnumOrigins[c]
	if origin == nil || !reflect.DeepEqual(c.Validation.Enum, []string{"1", "2"}) || len(origin.Members) != 3 || origin.Members[0].Kind() != document.Null || origin.Actual.(*ValueAlias).Type.Kind != Nullable {
		t.Fatal(origin)
	}
	if failure := a.validateNativeEnumBindings(ds); failure != nil {
		t.Fatal(failure)
	}
	copy, e := document.Load([]byte(`{"value":null}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	old := origin.Vocabulary.Items()[0]
	origin.Vocabulary.Items()[0] = get(copy, "value")
	if failure := a.validateNativeEnumBindings(ds); failure == nil {
		t.Fatal("Same-byte null node replacement authorized")
	}
	origin.Vocabulary.Items()[0] = old
	if failure := a.validateNativeEnumBindings(ds); failure != nil {
		t.Fatal(failure)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != string(input) {
		t.Fatal("Original null vocabulary rewritten")
	}
}
