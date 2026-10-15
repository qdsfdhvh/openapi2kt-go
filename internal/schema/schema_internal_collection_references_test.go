package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestSchemaInternalCollectionPointerFiniteIdentityAndRole(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Source":{"properties":{"leaf":{"type":"string"},"list":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/leaf"}},"map":{"additionalProperties":{"type":"array","items":{"type":"integer"}}},"cycle":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/cycle"}},"named":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"chain":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/ref"}},"ref":{"$ref":"#/components/schemas/Source/properties/leaf"}}},"Alias":{"type":"string"}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	schemas := get(get(root, "components"), "schemas")
	a := analyzer{root: root, components: map[string]*component{}}
	for _, name := range []string{"Source", "Alias"} {
		a.components[name] = &component{Schema: get(schemas, name), Wire: name, Pointer: "/components/schemas/" + name}
	}
	props := get(get(schemas, "Source"), "properties")
	for _, key := range []string{"list", "map", "chain"} {
		target, err := a.resolveSchema("#/components/schemas/Source/properties/"+key, "/use")
		if err != nil || target.Catalog != nil || target.Schema != get(props, key) || target.Pointer != "/components/schemas/Source/properties/"+key || len(a.components) != 2 {
			t.Fatal("borrowed collection acquired catalog identity", key, target, err)
		}
	}
	for _, key := range []string{"cycle", "named"} {
		_, err := a.resolveSchema("#/components/schemas/Source/properties/"+key, "/use")
		if err == nil || err.Code != "E_REF_UNSUPPORTED" || *err.Pointer != "/components/schemas/Source/properties/"+key+"/items" {
			t.Fatal("ref must terminate at internal scalar before planner", key, err)
		}
	}
	// A transparent scalar hop now borrows the actual terminal, with no catalog identity.
	target, err := a.resolveSchema("#/components/schemas/Source/properties/ref", "/use")
	if err != nil || target.Schema != get(props, "leaf") || target.Catalog != nil || target.Steps != 2 {
		t.Fatal(target, err)
	}
}
