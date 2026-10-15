package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestInternalDTOOriginalOwnerIdentity(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Owner":{"properties":{"node":{"nullable":true,"properties":{"id":{"type":"string"}}}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original := get(get(get(root, "components"), "schemas"), "Owner")
	catalog := &component{Name: "OwnerDto", Wire: "Owner", Schema: original, Pointer: "/components/schemas/Owner"}
	a := analyzer{root: root, options: Options{PackageName: "org.example.identity"}, components: map[string]*component{"Owner": catalog}}
	a.indexInternalModels(nil)
	target, err := a.resolveSchema("#/components/schemas/Owner/properties/node", "/use")
	if err != nil || target.Schema != get(get(original, "properties"), "node") || target.Catalog != nil || len(a.components) != 1 || target.ModelName != "org.example.identity.models.OwnerDto.Node" {
		t.Fatal("invented owner/source/catalog", target, err)
	}
	ty, err := a.referenceType(target, nil, 47, "/use")
	if err != nil || ty.Kind != Nullable || ty.Element.Name != target.ModelName {
		t.Fatal("root intent/barrier", ty, err)
	}
	no := false
	ty, err = a.referenceType(target, &no, 47, "/use")
	if err != nil || ty.Kind != Named {
		t.Fatal("false did not narrow canonical type", ty, err)
	}
	if _, err = a.referenceType(target, nil, 48, "/deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("dereference reset depth", err)
	}
}
