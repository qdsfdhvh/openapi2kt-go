package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestComposedUnionOriginalTerminalDeclarations(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/composed-union-terminals.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.products", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) || !a.nominalsLinked {
		t.Fatal("fabricated catalog or incomplete ownership")
	}
	retained := map[Declaration]bool{}
	var visit func(Declaration)
	var checkType func(TypeRef)
	checkType = func(ty TypeRef) {
		if ty.deferredNominal != nil {
			t.Fatal("unresolved semantic type", ty)
		}
		if ty.Element != nil {
			checkType(*ty.Element)
		}
	}
	visit = func(d Declaration) {
		retained[d] = true
		switch d := d.(type) {
		case *Model:
			for _, p := range d.Properties {
				checkType(p.Type)
			}
			for _, n := range d.Nested {
				visit(n)
			}
		case *ObjectUnion:
			for _, p := range d.Common {
				checkType(p.Type)
			}
			for _, v := range d.Variants {
				visit(v)
			}
		case *ValueAlias:
			checkType(d.Type)
		}
	}
	for _, d := range ds {
		visit(d)
	}
	for _, tc := range []struct {
		path, alias, owner, name string
		steps                    int
		nullable                 bool
	}{
		{"Hops/properties/root", "RootBridge", "Product", "ProductDto", 3, false},
		{"Hops/properties/nested", "NestedBridge", "Box", "BoxDto.Choice", 2, false},
		{"Hops/properties/narrow", "NarrowBridge", "Product", "ProductDto", 5, false},
		{"Box/properties/choice", "NestedAlias", "Box", "BoxDto.Choice", 1, false},
		{"Rows/items", "RowAlias", "Rows", "RowsDtoInner", 1, false},
		{"Map/additionalProperties", "MapAlias", "Map", "MapDtoValue", 1, false},
	} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		binding := a.deferredNominals[target.Schema]
		want := "org.example.products.models." + tc.name
		if binding == nil || binding.Schema != target.Schema || binding.Owner != a.components[tc.owner] || binding.Name != want || binding.Nullable || target.ModelName != want || target.Deferred != nil || max(1, target.Steps) != tc.steps {
			t.Fatal(tc, target, binding)
		}
		found := 0
		for d, origin := range a.unionOrigins {
			if retained[d] && origin.Schema == target.Schema {
				found++
				u, ok := d.(*ObjectUnion)
				if !ok || len(u.Variants) != 4 || origin.Pointer != target.Pointer || origin.Nullable {
					t.Fatal("original product union provenance", origin, target)
				}
				// Variants intentionally borrow this node. They cannot count as
				// additional physical owners of the product union itself.
				for _, v := range u.Variants {
					if a.modelOrigins[v] != target.Schema {
						t.Fatal("variant source copied")
					}
				}
			}
		}
		if found != 1 {
			t.Fatal("no unique retained product union", tc, found)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.Schema != target.Schema || alias.InternalModelTarget.ModelName != want || alias.InternalModelTarget.Deferred != nil {
			t.Fatal(alias)
		}
		a.aliasPlanning = true
		no := false
		ty, err := a.referenceType(target, &no, 48-tc.steps, "/source48")
		if err != nil || ty.Kind != Named || ty.Name != want || tc.steps > 1 && ty.SchemaSteps != tc.steps || ty.Slots() != 1 {
			t.Fatal("false lost classifier/source work", ty, err)
		}
		if _, err = a.referenceType(target, &no, 49-tc.steps, "/source49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
		a.aliasPlanning = false
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "BoxDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing product declaration authorized", err)
	}
	for _, name := range []string{"ProductMaybe", "ProductFalse", "NestedMaybe", "NestedFalse"} {
		c := a.components[name]
		if c.Schema != get(catalog, name) || c.BindingNullable == nil || *c.BindingNullable != (name == "ProductMaybe" || name == "NestedMaybe") {
			t.Fatal("root intent changed", c)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source mutated")
	}
}
