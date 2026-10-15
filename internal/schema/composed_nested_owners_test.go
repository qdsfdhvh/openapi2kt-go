package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestComposedNestedOwnersOriginalIdentityAndActualDeclarations(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/composed-nested-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.owners", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !a.nominalsLinked {
		t.Fatal("late ownership barrier open")
	}
	catalog := get(get(root, "components"), "schemas")
	if len(a.components) != len(catalog.Members()) {
		t.Fatal("catalog expanded")
	}
	refs := []struct{ alias, path, owner, name string }{{"ComposedNode", "Composed/allOf/0/properties/node", "Composed", "ComposedDto.Node"}, {"ChoiceNode", "Choice/oneOf/0/properties/node", "Choice", "ChoiceDto.First.Node"}, {"MixedNode", "MixedChoice/oneOf/1/properties/node", "MixedChoice", "ObjectMixedChoiceDto.Node"}, {"ComposedRow", "Composed/allOf/0/properties/rows/items", "Composed", "ComposedDto.RowsInner"}, {"RootRowNode", "RootRows/items/oneOf/0/properties/node", "RootRows", "RootRowsDtoInner.R.Node"}, {"RootMapNode", "RootMap/additionalProperties/anyOf/0/properties/node", "RootMap", "RootMapDtoValue.M.Node"}}
	for _, tc := range refs {
		target, err := a.lookupSchemaReference("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(err)
		}
		binding := a.deferredNominals[target.Schema]
		want := "org.example.owners.models." + tc.name
		if !target.ComposedLexical || target.SourceCatalog != a.components[tc.owner] || binding == nil || binding.Schema != target.Schema || binding.Owner != target.SourceCatalog || binding.Name != want || binding.Pointer != target.Pointer {
			t.Fatal(tc, binding, target)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.Schema != target.Schema || alias.InternalModelTarget.ModelName != want || alias.InternalModelTarget.Deferred != nil {
			t.Fatal("alias replaced or unresolved", tc, alias)
		}
		bound, err := a.resolveSchema("#/components/schemas/"+tc.path, "/late")
		if err != nil || bound.Schema != target.Schema || bound.ModelName != want || bound.Deferred != nil {
			t.Fatal("late identity", tc, bound, err)
		}
		no := false
		ty, err := a.referenceType(bound, &no, 47, "/height48")
		if err != nil || ty.Name != want || ty.Kind != Named {
			t.Fatal(ty, err)
		}
		if _, err = a.referenceType(bound, &no, 48, "/height49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
	}
	var visit func(Declaration, string)
	var checkType func(TypeRef)
	checkType = func(tref TypeRef) {
		if tref.deferredNominal != nil {
			t.Fatal("unresolved semantic type", tref)
		}
		if tref.Element != nil {
			checkType(*tref.Element)
		}
	}
	actual := map[string]*Model{}
	visit = func(d Declaration, parent string) {
		switch d := d.(type) {
		case *Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf, parent)
			}
		case *Model:
			actual[parent+"."+d.Name()] = d
			for _, p := range d.Properties {
				checkType(p.Type)
			}
			for _, n := range d.Nested {
				visit(n, parent+"."+d.Name())
			}
		case *ObjectUnion:
			for _, p := range d.Common {
				checkType(p.Type)
			}
			for _, v := range d.Variants {
				visit(v, parent+"."+d.Name())
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object, parent)
			}
		case *ValueAlias:
			checkType(d.Type)
		}
	}
	for _, d := range ds {
		visit(d, "org.example.owners.models")
	}
	for _, c := range a.components {
		checkType(c.AliasType)
	}
	for _, tc := range refs {
		binding := a.components[tc.alias].InternalModelTarget
		model := actual[binding.ModelName]
		if model == nil || a.modelOrigins[model] != binding.Schema || a.modelPointers[model] != binding.Pointer {
			t.Fatal("guessed classifier", tc)
		}
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "ComposedDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("borrowed copy authorized absent lexical owner", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source changed")
	}
}
