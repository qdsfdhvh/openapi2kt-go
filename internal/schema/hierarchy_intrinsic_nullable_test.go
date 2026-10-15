package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestIntrinsicHierarchyOriginalIntentAndPhysicalTargets(t *testing.T) {
	b, err := os.ReadFile("../../tests/fixtures/http/hierarchy-intrinsic-nullable.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(b, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	catalog := get(get(root, "components"), "schemas")
	before, err := catalog.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.hierarchyintrinsic", Suffix: "Dto"}, components: map[string]*component{}}
	decls, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"Root", "Group", "First"} {
		c := a.components[name]
		if c.Schema != get(catalog, name) || !c.NullableHierarchy || c.NullableDTO {
			t.Fatal(name, c)
		}
		ty := a.componentType(c)
		if ty.Kind != Nullable || ty.Element.Kind != Named || ty.Element.Name != "org.example.hierarchyintrinsic.models."+name+"Dto" {
			t.Fatal(name, ty)
		}
		target, e := a.lookupSchemaReference("#/components/schemas/"+name, "/use")
		if e != nil || !target.NullableBinding {
			t.Fatal(name, target, e)
		}
		no := false
		ty, e = a.referenceType(target, &no, 0, "/use")
		if e != nil || ty.Kind != Named {
			t.Fatal(name, ty, e)
		}
		found := false
		for _, d := range decls {
			if h, ok := d.(*Hierarchy); ok && h.ClassName == name+"Dto" {
				found = true
			}
		}
		if !found {
			t.Fatal("intrinsic source lacks actual physical hierarchy", name)
		}
	}
	for _, name := range []string{"ERoot", "EFirst", "EGroup"} {
		if a.components[name].BindingNullable == nil || !*a.components[name].BindingNullable {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"FRoot", "FFirst", "FGroup"} {
		if a.components[name].BindingNullable == nil || *a.components[name].BindingNullable {
			t.Fatal(name)
		}
	}
	after, err := catalog.MarshalJSON()
	if err != nil || string(before) != string(after) {
		t.Fatal("intrinsic intent changed source", err)
	}
}
