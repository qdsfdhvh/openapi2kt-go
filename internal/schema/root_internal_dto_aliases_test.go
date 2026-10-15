package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestRootInternalDTOAliasRetainsPhysicalNodeAndRealCatalog(t *testing.T) {
	input, readErr := os.ReadFile("../../tests/fixtures/http/root-internal-dto-aliases.json")
	if readErr != nil {
		t.Fatal(readErr)
	}
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{options: Options{PackageName: "org.example.roots", Suffix: "Dto"}, components: map[string]*component{}}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	schemas := get(get(root, "components"), "schemas")
	node := get(get(get(schemas, "Tree/key~"), "properties"), "node")
	alias := a.components["ChainAlias"]
	if len(a.components) != len(schemas.Members()) || alias.Schema != get(schemas, "ChainAlias") || alias.BindingTarget != nil || alias.InternalModelTarget.Schema != node || alias.InternalModelTarget.Pointer != "/components/schemas/Tree~1key~0/properties/node" || alias.AliasHeight != 2 {
		t.Fatal("fabricated catalog, copied node or lost root/internal source", alias)
	}
	if len(a.internalModelUses) != 4 || a.internalModelUses["org.example.roots.models.TreeKeyDto.Node"] == "" {
		t.Fatal("hierarchy reindex discarded root-alias provenance", a.internalModelUses)
	}
	target, err := a.lookupSchemaReference("#/components/schemas/ChainAlias", "/use")
	if err != nil || target.Catalog != alias || target.Schema != node || target.Steps != 3 || target.ModelName != "org.example.roots.models.TreeKeyDto.Node" {
		t.Fatal("named root binding disguised as borrowed terminal", target, err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("physical constructor owner not recognized", err)
	}
	if _, _, err := a.resolveNominal("#/components/schemas/ChainAlias", "/shape", 0); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("root alias leaked into nominal shape expansion", err)
	}
}
