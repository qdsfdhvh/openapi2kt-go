package schema

import "testing"

func TestInlineScalarCollectionChildFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Values":{"type":"array","items":{"type":"integer","minimum":0},"default":[1]},"Other":{"type":"array","items":{"type":"integer","minimum":0}},"Alias":{"$ref":"#/components/schemas/Values","default":[2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":[3]}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true},
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
	if source == nil || !source.InlineScalarCollection || len(source.CollectionFactories) != 2 {
		t.Fatal("missing inline collection factory proofs", source)
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Error("changed inline collection source retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored source rejected", name, failure)
		}
	}
	for _, name := range []string{"Values", "Alias"} {
		reject("factory "+name, func() func() {
			alias := componentDeclaration(declarations, a.components[name].Name).(*ValueAlias)
			old := alias.ComponentDefault
			alias.ComponentDefault = `[99]`
			return func() { alias.ComponentDefault = old }
		})
	}
	reject("same-type alias rebinding", func() func() {
		alias := a.components["Alias"]
		old := alias.BindingTarget
		alias.BindingTarget = a.components["Other"]
		return func() { alias.BindingTarget = old }
	})
	reject("copied terminal element", func() func() {
		alias := componentDeclaration(declarations, a.components["Values"].Name).(*ValueAlias)
		old := alias.Type.Element
		copied := *old
		alias.Type.Element = &copied
		return func() { alias.Type.Element = old }
	})
	reject("removed factory proof", func() func() {
		old := source.CollectionFactories
		source.CollectionFactories = source.CollectionFactories[:1]
		return func() { source.CollectionFactories = old }
	})
	reject("changed source role", func() func() {
		old := source.InlineScalarCollection
		source.InlineScalarCollection = false
		return func() { source.InlineScalarCollection = old }
	})
}
