package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestBridgeIntersectionBorrowsRealCatalogAndOriginalSource(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/bridge-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.bridge", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	declarations, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	catalog := get(get(root, "components"), "schemas")
	bridge := a.components["Bridge"]
	target := a.components["Value"]
	c, steps, intent, failure := a.resolveIntersectionMember("#/components/schemas/Bridge", "/use", 0)
	if failure != nil || c.Catalog != target || c.Schema != get(catalog, "Value") || steps != 4 || intent == nil || *intent || bridge.Schema != get(catalog, "Bridge") || len(a.components) != len(catalog.Members()) {
		t.Fatal("bridge invented catalog/source/nullability or lost cachedheight", c, steps, intent, failure)
	}
	s, failure := a.objectShape(get(catalog, "Consumer"), "/components/schemas/Consumer", 0)
	if failure != nil || shapeField(s, "id").Schema != get(get(c.Schema, "properties"), "id") || shapeField(s, "id").Pointer != "/components/schemas/Value/properties/id" {
		t.Fatal("borrowed fields lost originalnodes", failure)
	}
	if _, _, _, failure = a.resolveIntersectionMember("#/components/schemas/Bridge", "/49", 45); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("cachedbridgeheight reset", failure)
	}
	if _, _, failure = a.resolveNominal("#/components/schemas/Bridge", "/ordinary-shape", 0); failure == nil {
		t.Fatal("intersection permission leaked to other nominal roles")
	}
	if failure = a.validateInternalModels(declarations); failure != nil {
		t.Fatal(failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("AST changed")
	}
	// An original endpoint carrier is insufficient without its real catalog identity.
	saved := bridge.InternalModelTarget.Catalog
	fake := *saved
	bridge.InternalModelTarget.Catalog = &fake
	if _, _, _, failure = a.resolveIntersectionMember("#/components/schemas/Bridge", "/copied", 0); failure == nil {
		t.Fatal("copiedcatalog authorized shape")
	}
	bridge.InternalModelTarget.Catalog = saved
}
