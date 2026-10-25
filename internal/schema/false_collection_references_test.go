package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseCollectionOriginalSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Denied":false,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied","nullable":true}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsecollections", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	owner := a.components["Rows"].Schema
	copyOwner := get(get(get(copied, "components"), "schemas"), "Rows")
	for i, m := range owner.Members() {
		if m.Name == "items" {
			owner.Members()[i].Value = get(copyOwner, "items")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte false collection leaf source replacement authorized")
			}
			owner.Members()[i] = m
		}
	}
	leaf := get(owner, "items")
	copyLeaf := get(copyOwner, "items")
	for i, m := range leaf.Members() {
		if m.Name == "nullable" {
			leaf.Members()[i].Value = get(copyLeaf, "nullable")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte collection leaf nullable replacement authorized")
			}
			leaf.Members()[i] = m
		}
	}
	var alias *ValueAlias
	for _, d := range ds {
		if value, ok := d.(*ValueAlias); ok && value.ClassName == "RowsDto" {
			alias = value
		}
	}
	if alias == nil {
		t.Fatal("missing actual collection alias")
	}
	savedName := a.components["Rows"].Name
	a.components["Rows"].Name = "CopiedName"
	alias.ClassName = "CopiedName"
	if a.validateInternalModels(ds) == nil {
		t.Error("coherent collection classifier mutation authorized")
	}
	a.components["Rows"].Name = savedName
	alias.ClassName = savedName

	for _, tc := range []struct {
		name   string
		mutate func() func()
	}{
		{"height", func() func() {
			saved := a.components["Rows"].CollectionHeight
			a.components["Rows"].CollectionHeight++
			return func() { a.components["Rows"].CollectionHeight = saved }
		}},
		{"factory", func() func() { alias.ComponentDefault = "null"; return func() { alias.ComponentDefault = "" } }},
		{"backing", func() func() {
			saved := a.components["Rows"].AliasType
			a.components["Rows"].AliasType = TypeRef{Kind: JSON}
			return func() { a.components["Rows"].AliasType = saved }
		}},
		{"type", func() func() {
			saved := alias.Type
			alias.Type = TypeRef{Kind: JSON}
			return func() { alias.Type = saved }
		}},
		{"type-copy", func() func() {
			saved := alias.Type.Element
			cloned := *saved
			alias.Type.Element = &cloned
			return func() { alias.Type.Element = saved }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restore := tc.mutate()
			defer restore()
			if a.validateInternalModels(ds) == nil {
				t.Error("changed false collection authority admitted")
			}
		})
	}
	without := []Declaration{}
	for _, d := range ds {
		if d != alias {
			without = append(without, d)
		}
	}
	if a.validateInternalModels(without) == nil {
		t.Error("missing collection export authorized")
	}
	copiedAlias := *alias
	if a.validateInternalModels(append(without, &copiedAlias)) == nil {
		t.Error("same-byte copied collection export authorized")
	}
	if a.validateInternalModels(append(append([]Declaration{}, ds...), alias)) == nil {
		t.Error("duplicate collection export authorized")
	}

	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
