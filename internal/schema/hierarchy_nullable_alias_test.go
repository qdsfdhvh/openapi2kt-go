package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestHierarchyNullableAliasRetainsOriginalCatalogAndNominalBarrier(t *testing.T) {
	b, err := os.ReadFile("../../tests/fixtures/http/hierarchy-nullable-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(b, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	catalog := get(get(root, "components"), "schemas")
	aliasSource := get(catalog, "ARoot")
	physical := get(catalog, "Root")
	before, err := catalog.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.hierarchynullable", Suffix: "Dto"}, components: map[string]*component{}}
	declarations, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	alias := a.components["ARoot"]
	if alias.Schema != aliasSource || alias.BindingTarget != a.components["Root"] || alias.BindingTarget.Schema != physical || !a.hierarchySources[physical] || !a.hierarchySources[get(catalog, "First")] {
		t.Fatal(alias)
	}
	if alias.BindingNullable == nil || !*alias.BindingNullable || alias.AliasHeight != 1 {
		t.Fatal(alias)
	}
	target, e := a.lookupSchemaReference("#/components/schemas/BRoot", "/use")
	if e != nil || target.Schema != physical || target.Catalog != a.components["Root"] || target.Steps != 3 || !target.NullableBinding || target.Nullable == nil || !*target.Nullable {
		t.Fatal(target, e)
	}
	no := false
	ty, e := a.referenceType(target, &no, 45, "/use")
	if e != nil || ty.Kind != Named || ty.Name != "org.example.hierarchynullable.models.RootDto" {
		t.Fatal(ty, e)
	}
	if _, e := a.referenceType(target, &no, 46, "/deep"); e == nil || e.Code != "E_SCHEMA_DEPTH" {
		t.Fatal(e)
	}
	if a.hierarchySources[get(catalog, "AliasHolder")] {
		t.Fatal("ordinary holder mislabeled hierarchy")
	}
	found := false
	for _, d := range declarations {
		if h, ok := d.(*Hierarchy); ok && h.ClassName == "RootDto" {
			found = true
		}
	}
	if !found {
		t.Fatal("no final canonical hierarchy")
	}
	after, err := catalog.MarshalJSON()
	if err != nil || string(before) != string(after) {
		t.Fatal("catalog mutation", err)
	}
}
