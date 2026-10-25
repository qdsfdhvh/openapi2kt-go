package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseAliasActualFactoryAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsealiasdefaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var alias *ValueAlias
	for _, d := range ds {
		if value, ok := d.(*ValueAlias); ok {
			alias = value
		}
	}
	if alias == nil || alias.ComponentDefault != "null" {
		t.Fatal("missing own null factory")
	}
	alias.ComponentDefault = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("removed actual alias factory admitted")
	}
	alias.ComponentDefault = "null"
	proof := a.falseAliasFactories[alias]
	delete(a.falseAliasFactories, alias)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing actual factory authority admitted")
	}
	a.falseAliasFactories[alias] = proof
	savedType := alias.Type
	alias.Type = TypeRef{Kind: Nullable, Element: &TypeRef{Kind: JSON}}
	if a.validateInternalModels(ds) == nil {
		t.Error("nullable raw replacement codec authorized")
	}
	alias.Type = savedType
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	owner := a.components["Maybe"].Schema
	copyOwner := get(get(get(copyRoot, "components"), "schemas"), "Maybe")
	for i, m := range owner.Members() {
		if m.Name == "default" {
			owner.Members()[i].Value = get(copyOwner, "default")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte original null literal replaced")
			}
			owner.Members()[i] = m
		}
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
