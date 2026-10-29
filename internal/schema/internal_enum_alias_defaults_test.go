package schema

import "testing"

func TestInternalEnumFactoryOriginalAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Owner":{"properties":{"state":{"type":"string","format":"date","enum":["on","off"],"default":"on"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","nullable":true,"default":null}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	alias := componentDeclaration(ds, "AliasDto").(*ValueAlias)
	owner := a.components["Alias"]
	if alias.ComponentDefault != "null" || alias.Type.Kind != Nullable || len(a.internalEnumFactories.Order) != 1 {
		t.Fatal("independent actual factory missing")
	}
	proof := a.internalEnumFactories.Factories[alias]
	if proof.Terminal != get(get(a.components["Owner"].Schema, "properties"), "state") || proof.Value != get(owner.Schema, "default") || proof.Binding != owner.InternalModelTarget {
		t.Fatal("factory did not retain original node and typed binding")
	}
	checkRejected := func(name string, mutate func() func()) {
		t.Helper()
		undo := mutate()
		if err := a.validateInternalModels(ds); err == nil {
			t.Fatal("changed factory retained authority", name)
		}
		undo()
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal("restored factory failed", name, err)
		}
	}
	checkRejected("encoded annotation", func() func() {
		alias.ComponentDefault = `"on"`
		return func() { alias.ComponentDefault = "null" }
	})
	checkRejected("cached alias height", func() func() {
		owner.AliasHeight++
		return func() { owner.AliasHeight-- }
	})
	checkRejected("same-value binding copy", func() func() {
		original := owner.InternalModelTarget
		copy := *original
		owner.InternalModelTarget = &copy
		return func() { owner.InternalModelTarget = original }
	})
	checkRejected("actual nullable type", func() func() {
		original := alias.Type
		alias.Type = alias.Type.WithoutRootNull()
		return func() { alias.Type = original }
	})
	checkRejected("canonical declaration copy", func() func() {
		for i, d := range ds {
			if d == alias {
				copy := *alias
				ds[i] = &copy
				return func() { ds[i] = alias }
			}
		}
		t.Fatal("missing original alias")
		return nil
	})
	checkRejected("duplicate export", func() func() {
		ds = append(ds, alias)
		return func() { ds = ds[:len(ds)-1] }
	})
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("factory changed original schema/default nodes")
	}
}

func TestInternalEnumFactoryLookupSchemaPathsOnly(t *testing.T) {
	root := loadComponentDocument(t, `{"Owner":{"properties":{"state":{"enum":["on"]}},"default":{"data":{"enum":["on"]}}}}`)
	a := analyzer{componentSchemas: get(get(root, "components"), "schemas")}
	if _, leaf, ok := a.internalEnumFactoryLookup("#/components/schemas/Owner/properties/state"); !ok || leaf != get(get(get(a.componentSchemas, "Owner"), "properties"), "state") {
		t.Fatal("lookup lost original schema identity")
	}
	for _, ref := range []string{"#/components/schemas/Owner/default/data", "#/components/schemas/Owner/properties/state/enum/0", "#/components/schemas/Owner", "#/components/schemas/Missing/properties/state"} {
		if _, _, ok := a.internalEnumFactoryLookup(ref); ok {
			t.Fatal("non-schema data gained factory lookup permission", ref)
		}
	}
}
