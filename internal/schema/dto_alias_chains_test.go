package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOAliasChainOriginalCatalogAndSourceWork(t *testing.T) {
	for _, kind := range []string{"internal", "named"} {
		input, e := os.ReadFile("../../tests/fixtures/http/schema-" + kind + "-dto-alias-chains.json")
		if e != nil {
			t.Fatal(e)
		}
		root, loadErr := document.Load(input, document.JSON, "")
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		a := analyzer{options: Options{PackageName: "org.example.aliaschains", Suffix: "Dto"}, root: root, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		decls, err := a.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		schemas := get(get(root, "components"), "schemas")
		node := get(schemas, "Node")
		catalog := a.components["Node"]
		name := "org.example.aliaschains.models.NodeDto"
		if kind == "internal" {
			node = get(get(get(schemas, "Tree/key~"), "properties"), "node")
			catalog = a.components["NodeAliasChain"]
			name = "org.example.aliaschains.models.TreeKeyDto.Node"
		}
		count := len(a.components)
		a.aliasPlanning = true
		target, err := a.resolveSchema("#/components/schemas/Tree~1key~0/properties/nodeHop2", "/use")
		if err != nil || target.Schema != node || target.Catalog != catalog || target.Steps != 5 || target.Nullable == nil || *target.Nullable || len(a.components) != count {
			t.Fatal(kind, "lost canonical node/catalog/hops/nearestfalse", target, err)
		}
		if a.components["NodeAliasChain"].Schema != get(schemas, "NodeAliasChain") {
			t.Fatal("alias source replaced")
		}
		ty, err := a.referenceType(target, nil, 43, "/use")
		if err != nil || ty.Kind != Named || ty.Name != name || ty.SchemaSteps != 5 {
			t.Fatal(kind, "erased work", ty, err)
		}
		yes := true
		ty, err = a.referenceType(target, &yes, 43, "/use")
		if err != nil || ty.Kind != Nullable || ty.SchemaSteps != 5 {
			t.Fatal("nullable work", ty, err)
		}
		if _, err = a.referenceType(target, nil, 44, "/too-deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("cached consumer reset height", err)
		}
		if err = a.validateInternalModels(decls); err != nil {
			t.Fatal("physical shared owner", err)
		}
		borrowed, err := a.lookupSchemaReference("#/components/schemas/Tree~1key~0/properties/nodeHop2", "/borrowed")
		if err != nil {
			t.Fatal(err)
		}
		if resolved, err := a.resolveInternalReferenceChainAt(borrowed, "/borrowed", 0, map[*document.Value]bool{}, false); err != nil || resolved.Schema != node || resolved.Steps != target.Steps {
			t.Fatal("original ordinaryDTO leaf lost source/hops", resolved, err)
		}
		owner := "Node"
		if kind == "internal" {
			owner = "Tree/key~"
		}
		a.indexInternalModels(map[string]*Hierarchy{owner: {}})
		if _, err = a.resolveSchema("#/components/schemas/Tree~1key~0/properties/nodeHop2", "/hierarchy"); err == nil || err.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("hierarchy root admitted", err)
		}
	}
}
