package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestEnumNominalOriginalDeclarationIdentity(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/enum-nominal-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.enums", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) || !a.nominalsLinked {
		t.Fatal("invented catalog or open binding barrier")
	}
	retained := map[Declaration]bool{}
	var visit func(Declaration)
	visit = func(d Declaration) {
		retained[d] = true
		switch d := d.(type) {
		case *Model:
			for _, n := range d.Nested {
				visit(n)
			}
		case *ObjectUnion:
			for _, v := range d.Variants {
				visit(v)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		}
	}
	for _, d := range ds {
		visit(d)
	}
	for _, tc := range []struct {
		path, alias, owner, name string
		nullable                 bool
		steps                    int
	}{
		{"Owner/properties/state", "NestedState", "Owner", "OwnerDto.State", true, 1}, {"Owner/properties/rows/items", "RowState", "Owner", "OwnerDto.RowsInner", false, 1}, {"Choice/oneOf/0/properties/state", "VariantState", "Choice", "ChoiceDto.A.State", false, 1}, {"Hops/properties/next", "NestedBridge", "Owner", "OwnerDto.State", true, 4}, {"Hops/properties/named", "NamedBridge", "NamedState", "NamedStateDto", true, 3},
	} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		binding := a.deferredNominals[target.Schema]
		want := "org.example.enums.models." + tc.name
		if binding == nil || binding.Schema != target.Schema || binding.Owner != a.components[tc.owner] || binding.Name != want || binding.Nullable != tc.nullable || max(1, target.Steps) != tc.steps {
			t.Fatal(tc, target, binding)
		}
		found := false
		for d, origin := range a.enumOrigins {
			if retained[d] && origin.Schema == target.Schema {
				found = true
				if origin.Pointer != target.Pointer || origin.Nullable != tc.nullable {
					t.Fatal(origin, target)
				}
			}
		}
		if !found {
			t.Fatal("classifier not from retained enum", tc)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.Schema != target.Schema || alias.InternalModelTarget.ModelName != want || alias.InternalModelTarget.Deferred != nil {
			t.Fatal(alias)
		}
		no := false
		a.aliasPlanning = true
		ty, err := a.referenceType(target, &no, 48-tc.steps, "/source48")
		if err != nil || ty.Kind != Named || ty.Name != want || tc.steps > 1 && ty.SchemaSteps != tc.steps {
			t.Fatal(tc, ty, err)
		}
		if _, err = a.referenceType(target, &no, 49-tc.steps, "/source49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
		a.aliasPlanning = false
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "OwnerDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("absent actual owner authorized", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
