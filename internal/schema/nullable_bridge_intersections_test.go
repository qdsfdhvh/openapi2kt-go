package schema

import (
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"os"
	"testing"
)

func TestNullableBridgeOriginalDiscoveryAndFinalSourceAuthority(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/nullable-bridge-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.nullablebridge", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	// Nullability discovery operates before alias binding and grants no permission.
	for _, entry := range catalog.Members() {
		a.components[entry.Name] = &component{Wire: entry.Name, Name: entry.Name + "Dto", Pointer: naming.Pointer("/components/schemas", entry.Name), Schema: entry.Value}
	}
	consumer := a.components["Consumer"]
	for _, remaining := range []int{0, 16384} {
		budget := &compositionBudget{remaining: remaining, enabled: true}
		nullable, err := a.compositionNullable(consumer.Schema, consumer.Pointer, 0, budget, map[*document.Value]bool{})
		if remaining == 0 && (err == nil || err.Code != "E_SCHEMA_LIMIT") {
			t.Fatal("discovery reset budget", err)
		}
		if remaining > 0 && (err != nil || !nullable || budget.remaining != remaining-7) {
			t.Fatal("original internal-ref discovery lostnullable/work", nullable, err)
		}
	}
	if len(a.modelOrigins) != 0 || len(a.internalModels) != 0 || len(a.deferredNominals) != 0 {
		t.Fatal("discovery invented source authority")
	}
	a.components = map[string]*component{}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	bridge := a.components["Bridge"]
	// Binding/caching must not skip original named/internal alias work during
	// later nullability probes. The same seven original nodes remain charged.
	budget := &compositionBudget{remaining: 7, enabled: true}
	if nullable, failure := a.compositionNullable(get(catalog, "Consumer"), "/components/schemas/Consumer", 0, budget, map[*document.Value]bool{}); failure != nil || !nullable || budget.remaining != 0 {
		t.Fatal("bound aliases changed original discovery work", nullable, budget.remaining, failure)
	}
	budget = &compositionBudget{remaining: 6, enabled: true}
	if _, failure := a.compositionNullable(get(catalog, "Consumer"), "/components/schemas/Consumer", 0, budget, map[*document.Value]bool{}); failure == nil || failure.Code != "E_SCHEMA_LIMIT" {
		t.Fatal("bound alias cache bypassed shared work boundary", failure)
	}
	target := a.components["Value"]
	actual, steps, intent, err := a.resolveIntersectionMember("#/components/schemas/Bridge", "/use", 0)
	if err != nil || actual.Catalog != target || actual.Schema != get(catalog, "Value") || steps != 4 || intent == nil || !*intent || bridge.InternalModelTarget.Schema != target.Schema {
		t.Fatal("nullablebridge lost actualcatalog/source/nearestview", steps, intent, err)
	}
	s, err := a.objectShape(get(catalog, "Consumer"), "/components/schemas/Consumer", 0)
	if err != nil || s.Nullable == nil || !*s.Nullable || shapeField(s, "id").Schema != get(get(target.Schema, "properties"), "id") {
		t.Fatal("shape rewrote source/null intent", err)
	}
	if err = a.validateInternalModels(declarations); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("nullablediscovery mutated AST")
	}
	if _, _, _, err = a.resolveIntersectionMember("#/components/schemas/Bridge", "/source49", 45); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("nullablecachedhops reset", err)
	}
}
