package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestAllOfNominalTerminalsActualSourceAndCachedWork(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/allof-nominal-terminals.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.allof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.components) != len(catalog.Members()) {
		t.Fatal("fabricated catalog")
	}
	named := get(catalog, "AllModel")
	internal := get(get(get(catalog, "AllContainer"), "properties"), "node")
	for _, tc := range []struct {
		path, alias, name string
		node              *document.Value
		steps             int
		hasCatalog        bool
	}{{"AllHops/properties/named", "AllRootBridge", "AllModelDto", named, 2, true}, {"AllHops/properties/internal", "AllNestedBridge", "AllContainerDto.Node", internal, 2, false}} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil || target.Schema != tc.node || target.ModelName != "org.example.allof.models."+tc.name || target.Deferred != nil || target.Steps != tc.steps {
			t.Fatal(tc, target, err)
		}
		if tc.hasCatalog && target.Catalog != a.components["AllModel"] || !tc.hasCatalog && target.Catalog != nil {
			t.Fatal("original catalog terminal", target)
		}
		alias := a.components[tc.alias]
		if alias.Schema != get(catalog, tc.alias) || alias.InternalModelTarget.Schema != tc.node || alias.InternalModelTarget.Deferred != nil || alias.AliasHeight != tc.steps {
			t.Fatal("alias source/cacheheight", alias)
		}
		bound := a.deferredNominals[tc.node]
		if bound == nil || bound.Schema != tc.node || bound.Name != target.ModelName {
			t.Fatal("absent actual model binding", bound)
		}
		a.aliasPlanning = true
		no := false
		ty, err := a.referenceType(target, &no, 46, "/source48")
		if err != nil || ty.Kind != Named || ty.Name != bound.Name || ty.SchemaSteps != 2 {
			t.Fatal("erased internal/named steps", ty, err)
		}
		if _, err = a.referenceType(target, &no, 47, "/source49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
		aliasTarget, err := a.lookupSchemaReference("#/components/schemas/"+tc.alias, "/cached")
		if err != nil || aliasTarget.Catalog != alias || aliasTarget.Schema != tc.node || aliasTarget.Steps != 3 {
			t.Fatal(aliasTarget, err)
		}
		if _, err = a.referenceType(aliasTarget, &no, 46, "/cached49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("false reset alias cache", err)
		}
		a.aliasPlanning = false
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "AllContainerDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("absent actual composed classifier accepted", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source changed")
	}
}
