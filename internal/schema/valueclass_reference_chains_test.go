package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestValueClassInternalChainOriginalCanonicalLayout(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/valueclass-reference-chains.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.wrappers", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) || !a.nominalsLinked {
		t.Fatal("catalog identity changed")
	}
	retained := map[Declaration]bool{}
	for _, d := range ds {
		retained[d] = true
	}
	for _, tc := range []struct {
		path, alias, owner string
		nullable           bool
		steps              int
	}{{"Hops/properties/again", "CountBridge", "Count", true, 4}, {"Hops/properties/name", "NameBridge", "Name", true, 3}, {"Hops/properties/narrow", "CountBridgeFalse", "Count", false, 4}} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(err)
		}
		canonical := a.components[tc.owner]
		want := "org.example.wrappers.models." + canonical.Name
		if target.Catalog != canonical || target.Schema != get(catalog, tc.owner) || target.PhysicalValueClass != canonical || target.ModelName != want || target.Deferred != nil || target.Steps != tc.steps || target.Nullable == nil || *target.Nullable != tc.nullable {
			t.Fatal(tc, target)
		}
		found := false
		for d, origin := range a.valueClassOrigins {
			if retained[d] && origin.Schema == target.Schema {
				found = true
				if origin.Pointer != canonical.Pointer || origin.Nullable != canonical.NullableValueClass {
					t.Fatal("original wrapper provenance lost", origin)
				}
			}
		}
		if !found {
			t.Fatal("wrapper binding had no retained declaration")
		}
		c := a.components[tc.alias]
		if c.Schema != get(catalog, tc.alias) || c.InternalModelTarget.PhysicalValueClass != canonical {
			t.Fatal(c)
		}
		ty := a.componentType(c)
		raw := bare(ty)
		if raw.Kind != InlineValue || raw.Name != want || raw.Element == nil || raw.Element.Name != canonical.Scalar {
			t.Fatal("root lowered physical wrapper", tc, ty)
		}
		no := false
		ty, err = a.referenceType(target, &no, 48-tc.steps, "/depth48")
		raw = bare(ty)
		if err != nil || raw.Kind != InlineValue || raw.Element == nil || raw.Element.Name != canonical.Scalar {
			t.Fatal(tc, ty, err)
		}
		if raw.Slots() != a.componentType(canonical).WithoutRootNull().Slots() {
			t.Fatal("layout drift")
		}
		a.aliasPlanning = true
		ty, err = a.referenceType(target, &no, 48-tc.steps, "/cached48")
		if err != nil || ty.SchemaSteps != tc.steps {
			t.Fatal("false narrowing lost cached wrapper hops", ty, err)
		}
		a.aliasPlanning = false
		if _, err = a.referenceType(target, &no, 49-tc.steps, "/depth49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "CountDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing wrapper declaration authorized", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original schema mutated")
	}
}
