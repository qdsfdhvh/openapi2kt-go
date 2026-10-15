package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestRootDTOBridgeOriginalIdentityAndSourceWork(t *testing.T) {
	for _, kind := range []string{"named", "internal"} {
		input, e := os.ReadFile("../../tests/fixtures/http/root-" + kind + "-dto-bridges.json")
		if e != nil {
			t.Fatal(e)
		}
		root, loadErr := document.Load(input, document.JSON, "")
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		a := analyzer{options: Options{PackageName: "org.example.bridge", Suffix: "Dto"}, components: map[string]*component{}}
		declarations, err := a.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		schemas := get(get(root, "components"), "schemas")
		node := get(schemas, "Node")
		pointer := "/components/schemas/Node"
		name := "org.example.bridge.models.NodeDto"
		if kind == "internal" {
			node = get(get(get(schemas, "Tree/key~"), "properties"), "node")
			pointer = "/components/schemas/Tree~1key~0/properties/node"
			name = "org.example.bridge.models.TreeKeyDto.Node"
		}
		alias := a.components["NodeAlias"]
		if len(a.components) != len(schemas.Members()) || alias.Schema != get(schemas, "NodeAlias") || alias.BindingTarget != nil || alias.AliasHeight != 3 {
			t.Fatal("lost original root/catalog/sourceheight", kind, alias)
		}
		target := alias.InternalModelTarget
		if target == nil || target.Schema != node || target.Pointer != pointer || target.ModelName != name || target.Catalog == nil {
			t.Fatal("copied physical target", kind, target)
		}
		if kind == "named" && target.Catalog != a.components["Node"] {
			t.Fatal("fabricated named catalog")
		}
		if kind == "internal" && target.Catalog != a.components["ZOriginalNodeAlias"] {
			t.Fatal("lost real internal-alias catalog")
		}
		resolved, err := a.lookupSchemaReference("#/components/schemas/NodeAliasChain", "/use")
		if err != nil || resolved.Catalog != a.components["NodeAliasChain"] || resolved.Schema != node || resolved.ModelName != name || resolved.Steps != 5 {
			t.Fatal("cached mixed graph height/identity", resolved, err)
		}
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("not physical constructor-bearing", err)
		}
		if _, _, err := a.resolveNominal("#/components/schemas/NodeAlias", "/shape", 0); err == nil || err.Code != "E_REF_UNSUPPORTED" {
			t.Fatal("root bridge widened shape", err)
		}
	}
}

func TestRootDTOBridgeHierarchyRegistryRecheck(t *testing.T) {
	input, e := os.ReadFile("../../tests/fixtures/http/root-named-dto-bridges.json")
	if e != nil {
		t.Fatal(e)
	}
	root, loadErr := document.Load(input, document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	a := analyzer{options: Options{PackageName: "org.example.bridge"}, components: map[string]*component{}}
	_, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the second index's exclusion independently; the core contract
	// additionally supplies a real hierarchy fixture to the full analyzer.
	a.indexInternalModels(map[string]*Hierarchy{"Node": {}})
	node := a.components["Node"].Schema
	if a.physicalDTOName(node) != "" {
		t.Fatal("hierarchy owner still indexed")
	}
	target, err := a.lookupSchemaReference("#/components/schemas/NodeAlias", "/use")
	if err != nil || a.bindNamedChainModel(target, "/use") {
		t.Fatal("root bridge retained pruned owner", err)
	}
}
