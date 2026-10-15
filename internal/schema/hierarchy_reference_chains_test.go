package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestNamedHierarchyInternalChainsOriginalIdentityAndFinalOwnership(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/hierarchy-reference-chains.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	catalog := get(get(root, "components"), "schemas")
	before, e := catalog.MarshalJSON()
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.chains", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	a.aliasPlanning = true
	node := get(catalog, "Root")
	target, err := a.resolveSchema("#/components/schemas/Hop~1key~0/properties/parentSecond", "/use")
	if err != nil || target.Schema != node || target.Catalog != a.components["Root"] || target.Pointer != "/components/schemas/Root" || target.Steps != 6 || target.Nullable == nil || !*target.Nullable || len(a.components) != len(catalog.Members()) {
		t.Fatal(target, err)
	}
	no := false
	ty, err := a.referenceType(target, &no, 42, "/use")
	if err != nil || ty.Kind != Named || ty.Name != "org.example.chains.models.RootDto" || ty.SchemaSteps != 6 {
		t.Fatal(ty, err)
	}
	if _, err = a.referenceType(target, &no, 43, "/deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal(err)
	}
	c := a.components["ParentBridgeChain"]
	if c.Schema != get(catalog, "ParentBridgeChain") || c.InternalModelTarget.Schema != node || c.AliasHeight != 6 || c.BindingTarget != nil {
		t.Fatal(c)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	without := []Declaration{}
	for _, d := range ds {
		if _, ok := d.(*Hierarchy); !ok {
			without = append(without, d)
		}
	}
	if err = a.validateInternalModels(without); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("syntax invented hierarchy owner", err)
	}
	borrowed, err := a.lookupSchemaReference("#/components/schemas/Hop~1key~0/properties/parentSecond", "/borrowed")
	if err != nil {
		t.Fatal(err)
	}
	// This original input now graduates under the distinct borrowed hierarchy
	// profile; ordinary builtin admission does not invent the physical owner.
	bound, err := a.resolveInternalReferenceChainAt(borrowed, "/borrowed", 0, map[*document.Value]bool{}, false)
	if err != nil || bound.Schema != node || bound.Catalog != a.components["Root"] || a.physicalDTOName(bound.Schema) != "org.example.chains.models.RootDto" || a.borrowedHierarchyLeaves[node] == nil {
		t.Fatal("borrowed hierarchy lost actual named canonical source", bound, err)
	}
	after, e := catalog.MarshalJSON()
	if e != nil || string(before) != string(after) {
		t.Fatal("changed source")
	}
}
