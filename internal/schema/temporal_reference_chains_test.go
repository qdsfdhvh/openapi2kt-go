package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestTemporalReferenceChainsOriginalScalarAndCachedWork(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/temporal-reference-chains.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.temporal", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
		path, alias, canonical string
		steps                  int
		nullable               bool
	}{
		{"Hops/properties/stamp", "StampBridge", "Stamp", 3, true},
		{"Hops/properties/falseStamp", "StampFalseBridge", "Stamp", 4, false},
		{"Hops/properties/day", "DayBridge", "Day", 3, true},
		{"Hops/properties/native", "NativeBridge", "Native", 3, false},
		{"Hops/properties/again", "AgainBridge", "Stamp", 6, false},
	} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		canonical := a.components[tc.canonical]
		if target.CanonicalTemporal != canonical || target.Schema != canonical.Schema || target.Schema != get(catalog, tc.canonical) || target.Pointer != canonical.Pointer || target.ModelName != "" || target.Deferred != nil || target.Steps != tc.steps || target.Nullable != nil && *target.Nullable != tc.nullable {
			t.Fatal(tc, target)
		}
		if nullable, err := target.effectiveNullable(); err != nil || nullable != tc.nullable {
			t.Fatal("effective temporal intent changed", tc, nullable, err)
		}
		if tc.path == "Hops/properties/again" && target.Catalog != a.components["StampFalseBridge"] {
			t.Fatal("rootbridge catalog fabricated", target)
		}
		if a.deferredNominals[target.Schema] != nil {
			t.Fatal("transparent temporal gained nominal barrier")
		}
		found := 0
		for d, origin := range a.temporalOrigins {
			if retained[d] && origin.Owner == canonical {
				found++
				if origin.Schema != target.Schema || origin.Pointer != target.Pointer || origin.Scalar != canonical.Scalar || !temporalDeclaration(d, origin) {
					t.Fatal("canonical provenance lost", origin)
				}
			}
		}
		if found != 1 {
			t.Fatal("no unique actual transparent source declaration", found)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.CanonicalTemporal != canonical || alias.AliasHeight != tc.steps {
			t.Fatal(alias)
		}
		ty := a.componentType(alias)
		if bare(ty).Kind != Scalar || bare(ty).Name != canonical.Scalar || (ty.Kind == Nullable) != tc.nullable || ty.Slots() != 1 {
			t.Fatal("not canonical scalar", ty)
		}
		a.aliasPlanning = true
		no := false
		ty, err = a.referenceType(target, &no, 48-tc.steps, "/source48")
		if err != nil || ty.Kind != Scalar || ty.Name != canonical.Scalar || ty.SchemaSteps != tc.steps {
			t.Fatal("false erased native/source identity", ty, err)
		}
		if _, err = a.referenceType(target, &no, 49-tc.steps, "/source49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
		cached, err := a.lookupSchemaReference("#/components/schemas/"+tc.alias, "/cache")
		if err != nil || cached.Catalog != alias || cached.CanonicalTemporal != canonical || cached.Steps != tc.steps+1 {
			t.Fatal("cached catalog/hop drift", cached, err)
		}
		ty, err = a.referenceType(cached, &no, 47-tc.steps, "/cache48")
		if err != nil || ty.Kind != Scalar || ty.SchemaSteps != tc.steps+1 {
			t.Fatal(ty, err)
		}
		if _, err = a.referenceType(cached, &no, 48-tc.steps, "/cache49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("cached false reset height", err)
		}
		a.aliasPlanning = false
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "StampDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateTemporalBindings(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("missing canonical source declaration authorized", err)
	}
	canonical := a.components["Stamp"]
	for d, origin := range a.temporalOrigins {
		if origin.Owner != canonical {
			continue
		}
		copy := &ValueAlias{ClassName: d.Name(), Type: TypeRef{Kind: Scalar, Name: canonical.Scalar}.Null()}
		substituted := append([]Declaration{}, missing...)
		substituted = append(substituted, copy)
		if err = a.validateTemporalBindings(substituted); err == nil {
			t.Fatal("identically named replacement authorized without original declaration identity")
		}
		a.temporalOrigins[copy] = origin
		if err = a.validateTemporalBindings(append(append([]Declaration{}, ds...), copy)); err == nil {
			t.Fatal("multiple retained temporal declarations authorized")
		}
		delete(a.temporalOrigins, copy)
	}
	sourceSchema := canonical.Schema
	canonical.Schema = a.components["Day"].Schema
	if err = a.validateTemporalBindings(ds); err == nil {
		t.Fatal("copied/changed original temporal source authorized")
	}
	canonical.Schema = sourceSchema
	if err = a.validateTemporalBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source nodes mutated")
	}
}
