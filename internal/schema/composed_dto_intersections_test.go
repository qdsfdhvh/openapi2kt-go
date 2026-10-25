package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestComposedIntersectionUsesUniqueOriginalOccurrencesAndCapturedIdentity(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/composed-dto-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.composeddto", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) {
		t.Fatal("composed intersection fabricatedcatalog")
	}
	bytes := 0
	for _, tc := range []struct{ path, owner string }{{"Owner/allOf/0/properties/payload", "Owner"}, {"UnionOwner/oneOf/0/properties/payload", "UnionOwner"}} {
		target, steps, _, err := a.resolveIntersectionMember("#/components/schemas/"+tc.path, "/use", 0)
		if err != nil || target.Catalog != nil || target.SourceCatalog != a.components[tc.owner] || !target.ComposedLexical || steps != 1 {
			t.Fatal("actual composedsource lost", tc.path, target, err)
		}
		use := a.borrowedDTOLeaves[target.Schema]
		binding := a.deferredNominals[target.Schema]
		if use == nil || use.Actual == nil || binding == nil || binding.Owner != target.SourceCatalog || binding.Name != use.Name || binding.Schema != target.Schema || a.modelOrigins[use.Actual] != target.Schema || a.modelPointers[use.Actual] != target.Pointer {
			t.Fatal("source occurrence was guessed/copied", tc.path, use, binding)
		}
		b, _ := target.Schema.MarshalJSON()
		bytes += len(b)
		workBefore, bytesBefore := a.intersectionCaptures.Work, a.intersectionCaptures.Bytes
		context := a
		view := *target
		if _, _, _, err = context.internalDTOIntersection(&view, "/repeat", 0, 1, nil); err != nil || context.intersectionCaptures != a.intersectionCaptures || a.intersectionCaptures.Work != workBefore || a.intersectionCaptures.Bytes != bytesBefore {
			t.Fatal("composedcontext reset/recharged capture", err)
		}
		wrong := *target
		wrong.Pointer += "/properties/other"
		if _, _, _, err = a.internalDTOIntersection(&wrong, "/different-pointer", 0, 1, nil); err == nil {
			t.Fatal("cachedsame-schema authorizeddifferentpointer")
		}
		wrong = *target
		wrong.SourceCatalog = a.components["Hops"]
		if _, _, _, err = a.internalDTOIntersection(&wrong, "/foreign-real-owner", 0, 1, nil); err == nil {
			t.Fatal("cachedsame-schema authorizedforeigncatalog")
		}
		missing := []Declaration{}
		for _, d := range ds {
			if d.Name() != target.SourceCatalog.Name {
				missing = append(missing, d)
			}
		}
		if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("borrowedcopy authorizedmissingactualowner", tc.path, err)
		}
	}
	if len(a.intersectionCaptures.Seen) != 2 || a.intersectionCaptures.Bytes != bytes {
		t.Fatal("composedsource capture duplicated or skipped bytes", bytes, a.intersectionCaptures)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("composedsource AST changed")
	}
}
