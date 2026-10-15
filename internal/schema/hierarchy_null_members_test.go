package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestHierarchyNullMembersSourceIntentAndPhysicalEdges(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/hierarchy-null-members.json")
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
	a := analyzer{root: root, options: Options{PackageName: "org.example.nullmembers", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Root", "Group"} {
		c := a.components[name]
		if c.Schema != get(catalog, name) || !c.NullableHierarchy || has(c.Schema, "nullable") {
			t.Fatal("derived intent/source", c)
		}
		target, err := a.lookupSchemaReference("#/components/schemas/"+name, "/use")
		if err != nil || target.Nullable == nil || !*target.Nullable {
			t.Fatal(target, err)
		}
		no := false
		ty, err := a.referenceType(target, &no, 0, "/use")
		if err != nil || ty.Kind != Named {
			t.Fatal(ty, err)
		}
	}
	found := false
	for _, d := range ds {
		if h, ok := d.(*Hierarchy); ok && h.ClassName == "GroupDto" {
			found = true
			if h.Parent != "org.example.nullmembers.models.RootDto" {
				t.Fatal(h)
			}
		}
	}
	if !found {
		t.Fatal("physical hierarchy missing")
	}
	if a.components["ERoot"].BindingNullable == nil || !*a.components["ERoot"].BindingNullable {
		t.Fatal("derived alias intent lost")
	}
	after, e := catalog.MarshalJSON()
	if e != nil || string(before) != string(after) {
		t.Fatal("mutated source")
	}
}

func TestHierarchyNullMembersDiscoveryHasBoundedWork(t *testing.T) {
	// Repeated alias-member inspection counts even when source discovery is
	// provisional; invalid graphs cannot trigger unbounded fixed-point scans.
	a := analyzer{components: map[string]*component{}, hierarchySources: map[*document.Value]bool{}}
	b := []byte(`{"oneOf":[{"type":"null"}],"discriminator":{"propertyName":"tag"}}`)
	v, err := document.Load(b, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 16385 {
		name := string(rune(0x1000 + i))
		a.components[name] = &component{Wire: name, Pointer: "/components/schemas/" + name, Schema: v}
		a.hierarchySources[v] = true
	}
	if err := a.discoverHierarchyNullability(); err == nil || err.Code != "E_SCHEMA_LIMIT" {
		t.Fatal(err)
	}
}
