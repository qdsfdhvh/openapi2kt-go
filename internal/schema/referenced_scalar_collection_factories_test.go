package schema

import "testing"

func TestReferencedScalarCollectionFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":2},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":3},"default":[4]},"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[5]}}`)
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
		t.Fatal("finalize", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	if len(a.referencedScalarFactories) != 2 {
		t.Fatal("missing root or alias factory", len(a.referencedScalarFactories))
	}
	factory := a.referencedScalarFactories[0]
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Fatal("changed factory retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored factory rejected", name, failure)
		}
	}
	reject("canonical factory changed", func() func() {
		old := factory.Alias.ComponentDefault
		factory.Alias.ComponentDefault = "[9]"
		return func() { factory.Alias.ComponentDefault = old }
	})
	reject("same-name copied declaration", func() func() {
		old := factory.Alias
		copy := *old
		factory.Alias = &copy
		return func() { factory.Alias = old }
	})
	reject("factory type changed", func() func() {
		old := factory.Alias.Type
		factory.Alias.Type = TypeRef{Kind: List, Element: &TypeRef{Kind: Scalar, Name: "kotlin.Int"}}
		return func() { factory.Alias.Type = old }
	})
	reject("duplicate factory declaration", func() func() {
		declarations = append(declarations, factory.Alias)
		return func() { declarations = declarations[:len(declarations)-1] }
	})
}

func TestScalarCollectionFieldDefaultLateFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer","default":1},"Other":{"type":"integer","default":2},"Alias":{"$ref":"#/components/schemas/Leaf","default":3},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[4]}}}}`)
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
		t.Fatal("initial freeze", err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal("field default", err)
	}
	if err := a.validateFrozenCollectionReferenceChildDefaults(declarations, true); err != nil {
		t.Fatal("existing proof", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("late freeze", err)
	}
	if err := a.validateCollectionReferenceChildDefaults(declarations); err != nil {
		t.Fatal("late proof", err)
	}
	node := get(a.components["Rows"].Schema, "items")
	source := a.collectionChildDefaults[node]
	if source == nil || source.Value != nil || len(source.Factories) != 2 {
		t.Fatal("unannotated child did not bind independent factories", source)
	}
	retained := false
	for _, use := range source.Uses {
		retained = retained || use.Retained
	}
	if !retained {
		t.Fatal("late source proof did not retain the actual collection element")
	}
	alias := a.components["Alias"]
	old := alias.ScalarTarget
	alias.ScalarTarget = a.components["Other"]
	if err := a.validateCollectionReferenceChildDefaults(declarations); err == nil {
		t.Fatal("same-type alias rebinding retained authority")
	}
	alias.ScalarTarget = old
	rows := componentDeclaration(declarations, "RowsDto").(*ValueAlias)
	element := rows.Type.Element
	copy := *element
	rows.Type.Element = &copy
	if err := a.validateInternalModels(declarations); err == nil {
		t.Fatal("copied element retained authority")
	}
	rows.Type.Element = element
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("restored source rejected", err)
	}
}

func TestScalarCollectionFactoryBindsUnannotatedLeaf(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer"},"Other":{"type":"integer"},"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":[]}}`)
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
		t.Fatal("original source", err)
	}
	source := a.collectionChildDefaults[get(a.components["Rows"].Schema, "items")]
	if source == nil || source.Value != nil || len(source.Factories) != 0 {
		t.Fatal("missing unannotated leaf proof", source)
	}
	alias := a.components["Alias"]
	old := alias.ScalarTarget
	alias.ScalarTarget = a.components["Other"]
	if err := a.validateInternalModels(declarations); err == nil {
		t.Fatal("empty factory accepted a rebound scalar alias")
	}
	alias.ScalarTarget = old
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("restored source rejected", err)
	}
}

func TestScalarCollectionFactoryBindsIntermediateCollectionAliases(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer"},"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}},"Other":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}},"Middle":{"$ref":"#/components/schemas/Values"},"Alias":{"$ref":"#/components/schemas/Middle"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":[{"x":1}]}}`)
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
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Error("changed intermediate collection retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored authority rejected", name, failure)
		}
	}
	middle := a.components["Middle"]
	reject("same-type rebinding", func() func() {
		old := middle.BindingTarget
		middle.BindingTarget = a.components["Other"]
		return func() { middle.BindingTarget = old }
	})
	reject("alias height", func() func() {
		old := middle.AliasHeight
		middle.AliasHeight++
		return func() { middle.AliasHeight = old }
	})
	reject("collection height", func() func() {
		old := middle.CollectionHeight
		middle.CollectionHeight++
		return func() { middle.CollectionHeight = old }
	})
	reject("nullable binding", func() func() {
		old := middle.BindingNullable
		value := true
		middle.BindingNullable = &value
		return func() { middle.BindingNullable = old }
	})
	reject("replacement declaration", func() func() {
		for i, d := range declarations {
			if d.Name() == middle.Name {
				old := declarations[i]
				copy := *old.(*ValueAlias)
				declarations[i] = &copy
				return func() { declarations[i] = old }
			}
		}
		t.Fatal("missing intermediate declaration")
		return nil
	})
	reject("missing declaration", func() func() {
		for i, d := range declarations {
			if d.Name() == middle.Name {
				old := declarations
				declarations = append(append([]Declaration{}, declarations[:i]...), declarations[i+1:]...)
				return func() { declarations = old }
			}
		}
		t.Fatal("missing intermediate declaration")
		return nil
	})
	reject("duplicate declaration", func() func() {
		declarations = append(declarations, componentDeclaration(declarations, middle.Name))
		return func() { declarations = declarations[:len(declarations)-1] }
	})
	values := a.components["Values"]
	reject("canonical collection source", func() func() {
		old := values.Schema
		values.Schema = a.components["Other"].Schema
		return func() { values.Schema = old }
	})
	reject("canonical element replacement", func() func() {
		declaration := componentDeclaration(declarations, values.Name).(*ValueAlias)
		old := declaration.Type.Element
		copy := *old
		declaration.Type.Element = &copy
		return func() { declaration.Type.Element = old }
	})
}
