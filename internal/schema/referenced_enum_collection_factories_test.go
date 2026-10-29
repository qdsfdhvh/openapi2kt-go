package schema

import "testing"

func TestReferencedEnumCollectionFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Word":{"enum":["one","two","x"],"minLength":2,"default":"one"},"Tree":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["one"]},"Alias":{"$ref":"#/components/schemas/Tree","default":["two"]}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.factories", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if len(a.referencedEnumFactories) != 2 {
		t.Fatal("actual root/alias factory records missing")
	}
	alias := componentDeclaration(ds, "AliasDto").(*ValueAlias)
	reject := func(label string, mutate func() func()) {
		t.Helper()
		undo := mutate()
		if err := a.validateInternalModels(ds); err == nil {
			t.Fatal("changed factory retained permission", label)
		}
		undo()
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal("restored factory rejected", label, err)
		}
	}
	reject("initializer", func() func() {
		alias.ComponentDefault = `["one"]`
		return func() { alias.ComponentDefault = `["two"]` }
	})
	reject("missing records", func() func() {
		saved := a.referencedEnumFactories
		a.referencedEnumFactories = nil
		return func() { a.referencedEnumFactories = saved }
	})
	reject("duplicate export", func() func() { ds = append(ds, alias); return func() { ds = ds[:len(ds)-1] } })
	reject("canonical binding copy", func() func() {
		owner := a.components["Alias"]
		saved := owner.BindingTarget
		copy := *saved
		owner.BindingTarget = &copy
		return func() { owner.BindingTarget = saved }
	})
	reject("factory type", func() func() {
		saved := alias.Type.Name
		alias.Type.Name = "org.example.Other"
		return func() { alias.Type.Name = saved }
	})
	reject("linked backing copy", func() func() {
		saved := alias.Type.Element
		if saved == nil {
			t.Fatal("collection alias lost linked backing")
		}
		copy := *saved
		alias.Type.Element = &copy
		return func() { alias.Type.Element = saved }
	})
	reject("original source copy", func() func() {
		owner := a.components["Alias"]
		saved := owner.Schema
		copy := *saved
		owner.Schema = &copy
		return func() { owner.Schema = saved }
	})
	reject("cached height", func() func() {
		owner := a.components["Alias"]
		saved := owner.AliasHeight
		owner.AliasHeight++
		return func() { owner.AliasHeight = saved }
	})
	reject("duplicate factory proof", func() func() {
		saved := a.referencedEnumFactories
		a.referencedEnumFactories = append(a.referencedEnumFactories, saved[0])
		return func() { a.referencedEnumFactories = saved }
	})
	reject("retained source check", func() func() {
		checks := a.components["Word"].Validation
		saved := checks.MinLength
		value := int64(1)
		checks.MinLength = &value
		return func() { checks.MinLength = saved }
	})
}

func TestUncheckedNativeEnumCollectionFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Code":{"type":"integer","enum":[1,2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":[1]}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.factories", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	factory := a.referencedEnumFactories[0]
	if factory.Native == nil {
		t.Fatal("unchecked source lost actual native export proof")
	}
	reject := func(label string, mutate func() func()) {
		t.Helper()
		undo := mutate()
		if err := a.validateInternalModels(ds); err == nil {
			t.Fatal("changed unchecked source retained factory authority", label)
		}
		undo()
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal("restored unchecked source rejected", label, err)
		}
	}
	reject("missing native proof", func() func() {
		saved := factory.Native
		factory.Native = nil
		return func() { factory.Native = saved }
	})
	scalar := componentDeclaration(ds, "CodeDto").(*ScalarAlias)
	reject("scalar codec", func() func() {
		saved := scalar.Scalar
		scalar.Scalar = "kotlin.String"
		return func() { scalar.Scalar = saved }
	})
	reject("same-value export replacement", func() func() {
		saved := append([]Declaration{}, ds...)
		for i, d := range ds {
			if d == scalar {
				copy := *scalar
				ds[i] = &copy
			}
		}
		return func() { ds = saved }
	})
	reject("same-name extra export", func() func() {
		copy := *scalar
		ds = append(ds, &copy)
		return func() { ds = ds[:len(ds)-1] }
	})
}
