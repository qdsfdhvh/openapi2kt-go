package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestHierarchyNestedOwnersRetainedModelProvenanceAndNoDeferredTypes(t *testing.T) {
	b, e := os.ReadFile("../../tests/fixtures/http/hierarchy-nested-owners.json")
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
	a := analyzer{root: root, options: Options{PackageName: "org.example.owners", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	node := get(get(get(catalog, "First"), "properties"), "node")
	deferred := a.deferredNominals[node]
	if deferred == nil || deferred.Schema != node || deferred.Owner != a.components["First"] || deferred.Name != "org.example.owners.models.FirstDto.Node" || len(a.components) != len(catalog.Members()) {
		t.Fatal(deferred)
	}
	actual := false
	var check func(Declaration)
	var checkType func(TypeRef)
	checkType = func(ty TypeRef) {
		if ty.deferredNominal != nil {
			t.Fatal("unresolved semantic type", ty)
		}
		if ty.Element != nil {
			checkType(*ty.Element)
		}
	}
	check = func(d Declaration) {
		switch d := d.(type) {
		case *Hierarchy:
			if d.Leaf != nil {
				check(d.Leaf)
			}
		case *Model:
			if a.modelOrigins[d] == node {
				actual = true
				if d.ClassName != "Node" {
					t.Fatal(d)
				}
			}
			for _, p := range d.Properties {
				checkType(p.Type)
			}
			for _, n := range d.Nested {
				check(n)
			}
		case *ValueAlias:
			checkType(d.Type)
		}
	}
	for _, d := range ds {
		check(d)
	}
	if !actual {
		t.Fatal("binding authorized without retained model identity")
	}
	alias := a.components["NodeChain"]
	if alias.Schema != get(catalog, "NodeChain") || alias.InternalModelTarget.Schema != node || alias.InternalModelTarget.ModelName != deferred.Name || alias.InternalModelTarget.Deferred != nil {
		t.Fatal(alias)
	}
	target, err := a.resolveSchema("#/components/schemas/First/properties/node", "/use")
	if err != nil || target.Schema != node || target.SourceCatalog != a.components["First"] || !target.DirectLexical || target.ModelName != deferred.Name {
		t.Fatal(target, err)
	}
	no := false
	if _, err = a.referenceType(target, &no, 48, "/deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal(err)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	for _, d := range ds {
		if h, ok := d.(*Hierarchy); !ok || h.ClassName != "FirstDto" {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("absent physical hierarchy model authorized", err)
	}
	after, e := catalog.MarshalJSON()
	if e != nil || string(before) != string(after) {
		t.Fatal("source changed")
	}
}
