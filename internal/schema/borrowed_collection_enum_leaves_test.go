package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestBorrowedCollectionEnumLeavesActualVocabularyAndSourceAuthority(t *testing.T) {
	testBorrowedCollectionEnumLeavesAuthority(t, false)
}

func TestReferencedStringEnumUniqueActualVocabularyAndSourceAuthority(t *testing.T) {
	testBorrowedCollectionEnumLeavesAuthority(t, true)
}

func testBorrowedCollectionEnumLeavesAuthority(t *testing.T, unique bool) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"State":{"type":"string","enum":["ready","done"]},"Alias":{"$ref":"#/components/schemas/State"},"Owner":{"properties":{"state":{"type":"string","enum":["open","closed"]}}},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Owner/properties/state"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"MapBridge":{"$ref":"#/components/schemas/Source/properties/map"}}}}`)
	if unique {
		input = []byte(strings.Replace(string(input), `"type":"array","items"`, `"type":"array","uniqueItems":true,"items"`, 1))
	}
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedenums", Suffix: "Dto", Validation: unique}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var actual *StringEnum
	var owner *Model
	var without []Declaration
	for _, d := range ds {
		if d.Name() == "StateDto" {
			actual = d.(*StringEnum)
		} else {
			without = append(without, d)
		}
		if d.Name() == "OwnerDto" {
			owner = d.(*Model)
		}
	}
	if actual == nil || owner == nil || len(a.borrowedEnumLeaves) != 2 {
		t.Fatal("missing original enum source records")
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &StringEnum{ClassName: actual.ClassName, Members: actual.Members}), append(append([]Declaration{}, ds...), actual)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated original enum authorized")
		}
	}
	savedMember := actual.Members[0]
	actual.Members[0].Wire = "mutated"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed retained vocabulary authorized")
	}
	actual.Members[0] = savedMember
	originalOrigin := a.enumOrigins[actual]
	changedOrigin := originalOrigin
	changedOrigin.Nullable = !changedOrigin.Nullable
	a.enumOrigins[actual] = changedOrigin
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed actual enum source nullable view authorized")
	}
	a.enumOrigins[actual] = originalOrigin
	nested := owner.Nested
	owner.Nested = append(append([]Declaration{}, nested...), nested[0])
	if a.validateInternalModels(ds) == nil {
		t.Fatal("repeated nested actual enum authorized")
	}
	owner.Nested = nested
	original := a.components["State"].Schema
	enumArray := get(original, "enum")
	entry := enumArray.Items()[0]
	copyTree, copyErr := document.Load([]byte(`{"value":"ready"}`), document.JSON, "")
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	copyEntry := get(copyTree, "value")
	enumArray.Items()[0] = copyEntry
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte vocabulary entry node replacement authorized")
	}
	enumArray.Items()[0] = entry
	copied, err := document.Load([]byte(`{"type":"string","enum":["ready","done"]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a.components["State"].Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte copied enum source authorized")
	}
	a.components["State"].Schema = original
	alias := a.components["Alias"]
	aliasSource := alias.Schema
	copyAlias, err := document.Load([]byte(`{"$ref":"#/components/schemas/State"}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	alias.Schema = copyAlias
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte copied enum alias edge authorized")
	}
	alias.Schema = aliasSource
	height := alias.AliasHeight
	alias.AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed enum alias source cache authorized")
	}
	alias.AliasHeight = height
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(before) {
		t.Fatal("borrowed vocabulary mutated original tree")
	}
}
