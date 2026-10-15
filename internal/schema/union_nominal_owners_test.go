package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestUnionNominalOriginalDeclarationsAndNullViews(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/union-nominal-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.owners", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) || !a.nominalsLinked {
		t.Fatal("fabricated catalog or open late barrier")
	}
	refs := []struct {
		path, alias, owner, name string
		nullable                 bool
		steps                    int
	}{{"UnionBox/properties/choice", "UnionInner", "UnionBox", "UnionBoxDto.Choice", true, 1}, {"UnionBox/properties/native", "NativeInner", "UnionBox", "UnionBoxDto.Native", true, 1}, {"UnionHops/properties/object", "UnionRootBridge", "Choice", "ChoiceDto", false, 2}, {"UnionHops/properties/native", "UnionNativeBridge", "MixedChoice", "MixedChoiceDto", false, 2}}
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
		case *Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf)
			}
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
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *ValueAlias:
			checkType(d.Type)
		}
	}
	for _, d := range ds {
		visit(d)
	}
	for _, c := range a.components {
		checkType(c.AliasType)
	}
	for _, tc := range refs {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		binding := a.deferredNominals[target.Schema]
		want := "org.example.owners.models." + tc.name
		if binding == nil || binding.Schema != target.Schema || binding.Owner != a.components[tc.owner] || binding.Name != want || binding.Nullable != tc.nullable || target.ModelName != want || target.Deferred != nil || max(1, target.Steps) != tc.steps {
			t.Fatal(tc, target, binding)
		}
		found := false
		for d, origin := range a.unionOrigins {
			if retained[d] && origin.Schema == target.Schema {
				found = true
				if origin.Pointer != target.Pointer || origin.Nullable != tc.nullable {
					t.Fatal("original union provenance", origin, target)
				}
			}
		}
		if !found {
			t.Fatal("guessed union classifier", tc)
		}
		nullable, err := a.nominalSourceNullable(target)
		if err != nil || nullable != tc.nullable {
			t.Fatal("wrong emitted null view", tc, nullable, err)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.Schema != target.Schema || alias.InternalModelTarget.Deferred != nil || alias.InternalModelTarget.ModelName != want {
			t.Fatal(alias)
		}
		a.aliasPlanning = true
		no := false
		ty, err := a.referenceType(target, &no, 48-tc.steps, "/source48")
		if err != nil || ty.Kind != Named || ty.Name != want || tc.steps > 1 && ty.SchemaSteps != tc.steps {
			t.Fatal("false lost source work", ty, err)
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
		if d.Name() != "UnionBoxDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing actual union authorized", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source mutated")
	}
}
