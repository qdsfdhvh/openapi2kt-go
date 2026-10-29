package schema

import "testing"

func TestScalarReferenceChildDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer","minimum":0},"Other":{"type":"integer","minimum":0},"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":2}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.childdefault", Suffix: "Dto", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("finalize", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	node := get(a.components["Rows"].Schema, "items")
	source := a.collectionChildDefaults[node]
	if source == nil || len(source.Uses) == 0 || source.Uses[0].Actual == nil {
		t.Fatal("missing actual child source")
	}
	use := source.Uses[0]
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Fatal("changed scalar source retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored source rejected", name, failure)
		}
	}
	reject("alias rebound to another builtin", func() func() {
		alias := a.components["Alias"]
		old := alias.ScalarTarget
		alias.ScalarTarget = a.components["Other"]
		return func() { alias.ScalarTarget = old }
	})
	reject("source checks replaced", func() func() {
		old := use.Actual.Validation
		if old == nil {
			t.Fatal("missing original check")
		}
		copy := *old
		use.Actual.Validation = &copy
		return func() { use.Actual.Validation = old }
	})
	reject("actual element replaced", func() func() {
		rows := componentDeclaration(declarations, "RowsDto").(*ValueAlias)
		old := rows.Type.Element
		copy := *old
		rows.Type.Element = &copy
		return func() { rows.Type.Element = old }
	})
	reject("annotation value changed", func() func() {
		old := source.Encoded
		source.Encoded = "3"
		return func() { source.Encoded = old }
	})
}

func TestScalarReferenceChildDefaultRejectsEquivalentAliasRebinding(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer"},"Other":{"type":"integer"},"Inner":{"$ref":"#/components/schemas/Leaf"},"Alias":{"$ref":"#/components/schemas/Inner"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":2}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.childdefault", Suffix: "Dto", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("finalize", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	for _, name := range []string{"Alias", "Inner"} {
		alias := a.components[name]
		original := alias.ScalarTarget
		alias.ScalarTarget = a.components["Other"]
		if err := a.validateInternalModels(declarations); err == nil {
			t.Fatal("equivalent scalar root retained alias source authority", name)
		}
		alias.ScalarTarget = original
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("restored authority", name, err)
		}
	}
}

func TestScalarReferenceChildRetainsIndependentFactories(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":2},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":3}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.childdefault", Suffix: "Dto", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("finalize", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original factory authority", err)
	}
	node := get(a.components["Rows"].Schema, "items")
	source := a.collectionChildDefaults[node]
	if source == nil || len(source.Factories) != 2 {
		t.Fatal("target and alias factory bindings were not retained")
	}
	for _, factory := range source.Factories {
		var restore func()
		switch declaration := factory.Declaration.(type) {
		case *ScalarAlias:
			old := declaration.ComponentDefault
			declaration.ComponentDefault = "99"
			restore = func() { declaration.ComponentDefault = old }
		case *ValueAlias:
			old := declaration.ComponentDefault
			declaration.ComponentDefault = "99"
			restore = func() { declaration.ComponentDefault = old }
		default:
			t.Fatal("unexpected factory declaration", factory.Owner.Name)
		}
		if a.validateInternalModels(declarations) == nil {
			t.Fatal("changed factory value retained child authority", factory.Owner.Name)
		}
		restore()
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("restored factory rejected", factory.Owner.Name, err)
		}
	}
	factories := source.Factories
	source.Factories = nil
	if a.validateInternalModels(declarations) == nil {
		t.Fatal("dropped factory bindings retained child authority")
	}
	source.Factories = factories
	oldOwner := source.Factories[0].Owner
	source.Factories[0].Owner = source.Factories[1].Owner
	if a.validateInternalModels(declarations) == nil {
		t.Fatal("factory rebound to another owner retained child authority")
	}
	source.Factories[0].Owner = oldOwner
	for _, factory := range factories {
		var remaining []Declaration
		for _, declaration := range declarations {
			if declaration != factory.Declaration {
				remaining = append(remaining, declaration)
			}
		}
		if a.validateInternalModels(remaining) == nil {
			t.Fatal("missing factory declaration retained child authority", factory.Owner.Name)
		}
		if a.validateInternalModels(append(append([]Declaration{}, declarations...), factory.Declaration)) == nil {
			t.Fatal("duplicate factory declaration retained child authority", factory.Owner.Name)
		}
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("restored declarations rejected", err)
	}
}
