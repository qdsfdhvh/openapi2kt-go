package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableAllOfDTOOriginalModelAuthorityAndSource(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/nullable-allof-dtos.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := &analyzer{root: root, options: Options{PackageName: "org.example.nullableallof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	schemas := get(get(root, "components"), "schemas")
	if len(a.components) != len(schemas.Members()) {
		t.Fatal("synthetic catalog introduced")
	}
	c := a.components["ZRecord"]
	if c.Schema != get(schemas, "ZRecord") || !c.NullableDTO {
		t.Fatal(c)
	}
	var physical *Model
	for _, d := range ds {
		if d.Name() == "ZRecordDto" {
			physical = d.(*Model)
		}
	}
	if physical == nil || !physical.Nullable || len(physical.Properties) != 2 || a.modelOrigins[physical] != c.Schema || a.modelPointers[physical] != "/components/schemas/ZRecord" {
		t.Fatal("actual original constructor absent", physical)
	}
	shape, e := a.objectShape(c.Schema, c.Pointer, 0)
	if e != nil {
		t.Fatal(e)
	}
	id := shapeField(shape, "id")
	base := get(get(get(schemas, "Base"), "properties"), "id")
	if id.Schema != base || id.Pointer != "/components/schemas/Base/properties/id" || id.Depth != 2 {
		t.Fatal("copied composition field", id)
	}
	for _, name := range []string{"ZRecord", "AInherited", "AFalse"} {
		target, e := a.resolveSchema("#/components/schemas/"+name, "/use")
		if e != nil {
			t.Fatal(e)
		}
		ty, e := a.referenceType(target, nil, 0, "/use")
		if e != nil {
			t.Fatal(e)
		}
		nullable := name != "AFalse"
		if (ty.Kind == Nullable) != nullable || bare(ty).Name != "org.example.nullableallof.models.ZRecordDto" {
			t.Fatal(name, ty)
		}
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(before) {
		t.Fatal("source mutation")
	}
}
