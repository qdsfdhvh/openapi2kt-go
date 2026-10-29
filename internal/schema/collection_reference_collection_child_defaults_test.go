package schema

import "testing"

func TestCollectionReferenceChildRetainsIntermediateAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Leaf":{"type":"integer","minimum":0},"OtherLeaf":{"type":"integer","minimum":0},"LeafAlias":{"$ref":"#/components/schemas/Leaf"},"LeafAlias2":{"$ref":"#/components/schemas/LeafAlias"},"Values":{"additionalProperties":{"$ref":"#/components/schemas/LeafAlias2"}},"Other":{"additionalProperties":{"$ref":"#/components/schemas/LeafAlias2"}},"Middle":{"$ref":"#/components/schemas/Values"},"Alias":{"$ref":"#/components/schemas/Middle"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":{"child":2}}}}`)
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
	if source == nil || !source.Collection || !source.CollectionValueReady || len(source.Collections) < 3 {
		t.Fatal("collection source did not retain its chain", source)
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Error("changed collection child retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored source rejected", name, failure)
		}
	}
	middle := a.components["Middle"]
	reject("same-type intermediate rebinding", func() func() {
		old := middle.BindingTarget
		middle.BindingTarget = a.components["Other"]
		return func() { middle.BindingTarget = old }
	})
	for _, name := range []string{"LeafAlias", "LeafAlias2"} {
		reject("same-type scalar alias rebinding "+name, func() func() {
			alias := a.components[name]
			old := alias.ScalarTarget
			alias.ScalarTarget = a.components["OtherLeaf"]
			return func() { alias.ScalarTarget = old }
		})
	}
	reject("scalar alias checks replaced", func() func() {
		alias := a.components["LeafAlias"]
		old := alias.Validation
		if old == nil {
			t.Fatal("missing original scalar checks")
		}
		copy := *old
		alias.Validation = &copy
		return func() { alias.Validation = old }
	})
	reject("scalar component source replaced", func() func() {
		leaf := a.components["Leaf"]
		old := leaf.Schema
		leaf.Schema = a.components["OtherLeaf"].Schema
		return func() { leaf.Schema = old }
	})
	reject("scalar source proof removed", func() func() {
		old := source.ScalarSources
		source.ScalarSources = nil
		return func() { source.ScalarSources = old }
	})
	reject("cached collection height", func() func() {
		old := middle.CollectionHeight
		middle.CollectionHeight++
		return func() { middle.CollectionHeight = old }
	})
	reject("cached alias height", func() func() {
		old := middle.AliasHeight
		middle.AliasHeight++
		return func() { middle.AliasHeight = old }
	})
	reject("nullable binding", func() func() {
		old := middle.BindingNullable
		value := true
		middle.BindingNullable = &value
		return func() { middle.BindingNullable = old }
	})
	reject("copied intermediate declaration", func() func() {
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
	reject("missing intermediate declaration", func() func() {
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
	reject("duplicate intermediate declaration", func() func() {
		declarations = append(declarations, componentDeclaration(declarations, middle.Name))
		return func() { declarations = declarations[:len(declarations)-1] }
	})
	reject("copied canonical element", func() func() {
		alias := componentDeclaration(declarations, a.components["Values"].Name).(*ValueAlias)
		old := alias.Type.Element
		copied := *old
		alias.Type.Element = &copied
		return func() { alias.Type.Element = old }
	})
	reject("changed annotation", func() func() {
		old := source.Encoded
		source.Encoded = `{"child":3}`
		return func() { source.Encoded = old }
	})
}
