package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestBorrowedCollectionDTOLeavesActualUniqueConstructorAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Node":{"properties":{"id":{"type":"string"}},"required":["id"]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.borroweddto", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	var original *Model
	var without []Declaration
	for _, d := range ds {
		if d.Name() == "NodeDto" {
			original = d.(*Model)
		} else {
			without = append(without, d)
		}
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &Model{ClassName: original.ClassName, Properties: original.Properties}), append(append([]Declaration{}, ds...), original)} {
		if err := a.validateInternalModels(bad); err == nil {
			t.Fatal("missing/replaced/repeated actual borrowed DTO leaf authorized")
		}
	}
	saved := a.components["Node"].Schema
	copied, err := document.Load([]byte(`{"properties":{"id":{"type":"string"}},"required":["id"]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a.components["Node"].Schema = copied
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("same-byte copied borrowed nominal source authorized")
	}
	a.components["Node"].Schema = saved
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	// A referenced named alias is a source edge, not just its cached terminal.
	aliasInput := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Node":{"properties":{"id":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Node"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}}}`)
	root2, err := document.Load(aliasInput, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	b := analyzer{root: root2, options: a.options, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	aliases, err := b.analyze(root2)
	if err != nil {
		t.Fatal(err)
	}
	aliasSource := b.components["Alias"].Schema
	copiedAlias, err := document.Load([]byte(`{"$ref":"#/components/schemas/Node"}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	b.components["Alias"].Schema = copiedAlias
	if b.validateInternalModels(aliases) == nil {
		t.Fatal("same-byte named alias edge replacement authorized")
	}
	b.components["Alias"].Schema = aliasSource
}
