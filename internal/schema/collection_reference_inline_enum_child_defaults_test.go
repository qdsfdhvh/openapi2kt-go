package schema

import "testing"

func TestInlineEnumCollectionChildAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Values":{"type":"array","items":{"type":"string","enum":["one","two"]},"default":["one"]},"Other":{"type":"array","items":{"type":"string","enum":["one","two"]}},"Alias":{"$ref":"#/components/schemas/Values","default":["two"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":["one"]}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.defaults", Suffix: "Dto"},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("freeze", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	source := a.collectionChildDefaults[get(a.components["Rows"].Schema, "items")]
	if source == nil || !source.InlineEnumCollection || len(source.CollectionFactories) != 2 {
		t.Fatal("missing inline enum collection factory proofs", source)
	}
	leaf := get(a.components["Values"].Schema, "items")
	var actual *StringEnum
	for _, declaration := range declarations {
		if candidate, ok := declaration.(*StringEnum); ok && a.enumOrigins[candidate].Schema == leaf {
			actual = candidate
		}
	}
	if actual == nil || len(actual.Members) != 2 {
		t.Fatal("missing inline enum peer")
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Error("changed inline enum source retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored source rejected", name, failure)
		}
	}
	reject("changed peer vocabulary", func() func() {
		old := actual.Members[0]
		actual.Members[0].Wire = "future"
		return func() { actual.Members[0] = old }
	})
	reject("changed peer origin", func() func() {
		old := a.enumOrigins[actual]
		changed := old
		changed.Pointer += "/copy"
		a.enumOrigins[actual] = changed
		return func() { a.enumOrigins[actual] = old }
	})
	reject("removed peer snapshot", func() func() {
		old := source.InlineEnumPeerSnapshot
		source.InlineEnumPeerSnapshot = nil
		return func() { source.InlineEnumPeerSnapshot = old }
	})
	reject("same-type alias rebinding", func() func() {
		alias := a.components["Alias"]
		old := alias.BindingTarget
		alias.BindingTarget = a.components["Other"]
		return func() { alias.BindingTarget = old }
	})
	reject("changed target factory", func() func() {
		alias := componentDeclaration(declarations, a.components["Values"].Name).(*ValueAlias)
		old := alias.ComponentDefault
		alias.ComponentDefault = `["future"]`
		return func() { alias.ComponentDefault = old }
	})
	withoutPeer := []Declaration{}
	for _, declaration := range declarations {
		if declaration != actual {
			withoutPeer = append(withoutPeer, declaration)
		}
	}
	copied := *actual
	copied.Members = append([]EnumMember{}, actual.Members...)
	for _, invalid := range [][]Declaration{withoutPeer, append(append([]Declaration{}, withoutPeer...), &copied), append(append([]Declaration{}, declarations...), actual)} {
		if a.validateInternalModels(invalid) == nil {
			t.Error("missing, copied or duplicated enum peer retained authority")
		}
	}
}
