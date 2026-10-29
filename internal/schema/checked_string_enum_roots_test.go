package schema

import "testing"

func TestCheckedStringEnumRootGetterWithForwardCollection(t *testing.T) {
	root := loadComponentDocument(t, `{"Word":{"type":["string","null"],"enum":["one","two","x"],"minLength":2,"default":"one"},"Alias":{"$ref":"#/components/schemas/Word","nullable":false,"default":"two"},"Tree":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}},"Holder":{"properties":{"word":{"$ref":"#/components/schemas/Alias","default":"one"},"tree":{"$ref":"#/components/schemas/Tree","default":{"field":["one"]}}}}}`)
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, referencedDefaults: &defaults, options: Options{PackageName: "org.example.checked", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal("valid original getter lost to provisional collection backing", err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	ty := &holder.Properties[1].Type
	for ty.Kind == Nullable {
		ty = ty.Element
	}
	backing := ty.Element
	if backing == nil {
		t.Fatal("missing linked collection backing")
	}
	copy := *backing
	ty.Element = &copy
	if err := a.validateInternalModels(ds); err == nil {
		t.Fatal("same-value linked backing replacement retained getter authority")
	}
	ty.Element = backing
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal("restored linked getter rejected", err)
	}
}

func TestCheckedStringEnumRootAuthority(t *testing.T) {
	for _, withConsumers := range []bool{false, true} {
		body := `"Word":{"enum":["one","two","x"],"minLength":2,"default":"one"},"Alias":{"$ref":"#/components/schemas/Word","default":"two"}`
		if withConsumers {
			body += `,"Use":{"properties":{"word":{"$ref":"#/components/schemas/Alias"}}}`
		}
		root := loadComponentDocument(t, `{`+body+`}`)
		a := analyzer{root: root, options: Options{PackageName: "org.example.checked", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		word := a.components["Word"]
		enum := componentDeclaration(ds, "WordDto").(*StringEnum)
		alias := componentDeclaration(ds, "AliasDto").(*ValueAlias)
		checked := a.checkedEnumSources[word.Schema]
		if checked.Root != word || !checked.Declarations[enum] || len(checked.RootFactories) != 1 || word.Validation != checked.Checks {
			t.Fatal("root/alias lost actual declaration/source/check identity")
		}
		reject := func(label string, mutate func() func()) {
			t.Helper()
			undo := mutate()
			if err := a.validateInternalModels(ds); err == nil {
				t.Fatal("changed root retained authority", label)
			}
			undo()
			if err := a.validateInternalModels(ds); err != nil {
				t.Fatal("restored root rejected", label, err)
			}
		}
		reject("root check", func() func() {
			saved := word.Validation
			word.Validation = nil
			return func() { word.Validation = saved }
		})
		reject("vocabulary", func() func() {
			saved := enum.Members[0]
			enum.Members[0].Wire = "changed"
			return func() { enum.Members[0] = saved }
		})
		reject("root factory", func() func() { enum.ComponentDefault = `"two"`; return func() { enum.ComponentDefault = `"one"` } })
		reject("alias factory", func() func() { alias.ComponentDefault = `"one"`; return func() { alias.ComponentDefault = `"two"` } })
		reject("missing factory registry", func() func() {
			saved := checked.RootFactories
			checked.RootFactories = nil
			return func() { checked.RootFactories = saved }
		})
		reject("duplicate root", func() func() { ds = append(ds, enum); return func() { ds = ds[:len(ds)-1] } })
		reject("duplicate factory", func() func() { ds = append(ds, alias); return func() { ds = ds[:len(ds)-1] } })
		reject("canonical binding", func() func() {
			owner := a.components["Alias"]
			saved := owner.BindingTarget
			copy := *saved
			owner.BindingTarget = &copy
			return func() { owner.BindingTarget = saved }
		})
		if withConsumers {
			reject("actual consumer check", func() func() {
				use := componentModel(ds, "UseDto")
				ty := &use.Properties[0].Type
				for ty.Kind == Nullable {
					ty = ty.Element
				}
				saved := ty.Validation
				ty.Validation = nil
				return func() { ty.Validation = saved }
			})
		}
	}
}
