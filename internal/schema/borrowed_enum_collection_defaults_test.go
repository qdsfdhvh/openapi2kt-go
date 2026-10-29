package schema

import "testing"

func TestInternalEnumCollectionFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Owner":{"properties":{"tree":{"type":"array","items":{"enum":["one","two","x"],"minLength":2},"default":["one"]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/tree","default":["two"]}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	alias := componentDeclaration(ds, "AliasDto").(*ValueAlias)
	owner := a.components["Alias"]
	proof := a.internalEnumFactories.Factories[alias]
	if proof == nil || proof.Terminal != owner.InternalModelTarget.BorrowedCollection.Schema || alias.ComponentDefault != `["two"]` {
		t.Fatal("collection factory lost actual original source/backing")
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		undo := mutate()
		if err := a.validateInternalModels(ds); err == nil {
			t.Fatal("changed collection factory retained permission", name)
		}
		undo()
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal("restored collection factory rejected", name, err)
		}
	}
	reject("annotation", func() func() {
		alias.ComponentDefault = `["one"]`
		return func() { alias.ComponentDefault = `["two"]` }
	})
	reject("binding identity", func() func() {
		original := owner.InternalModelTarget
		copy := *original
		owner.InternalModelTarget = &copy
		return func() { owner.InternalModelTarget = original }
	})
	reject("factory export", func() func() {
		ds = append(ds, alias)
		return func() { ds = ds[:len(ds)-1] }
	})
	reject("actual backing check", func() func() {
		leaf := &owner.InternalModelTarget.BorrowedCollection.Type
		for leaf.Element != nil {
			leaf = leaf.Element
		}
		check := leaf.Validation
		leaf.Validation = nil
		return func() { leaf.Validation = check }
	})
}
