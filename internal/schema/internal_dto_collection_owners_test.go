package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestInternalDTOCollectionPhysicalIdentity(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Root":{"type":"array","nullable":true,"items":{"additionalProperties":{"properties":{"children":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}},"additionalProperties":{"properties":{"ignored":{"type":"string"}}}}}},"Alias":{"$ref":"#/components/schemas/Root"}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	schemas := get(get(root, "components"), "schemas")
	schema := get(schemas, "Root")
	c := &component{Wire: "Root", Name: "RootDto", Schema: schema, Pointer: "/components/schemas/Root"}
	a := analyzer{options: Options{PackageName: "org.example.collections"}, components: map[string]*component{"Root": c, "Alias": {Wire: "Alias", Name: "AliasDto", Schema: get(schemas, "Alias"), Pointer: "/components/schemas/Alias", BindingTarget: c}}}
	a.indexInternalModels(nil)
	parent := get(get(schema, "items"), "additionalProperties")
	child := get(get(get(parent, "properties"), "children"), "items")
	if len(a.internalModels) != 2 || a.internalModels[parent] != "org.example.collections.models.RootDtoInnerValue" || a.internalModels[child] != "org.example.collections.models.RootDtoInnerValue.ChildrenInner" {
		t.Fatal("physical scope/Inner/Value/alias/scratch", a.internalModels)
	}
	target, err := a.resolveSchema("#/components/schemas/Root/items/additionalProperties/properties/children/items", "/use")
	if err != nil || target.Schema != child || target.Catalog != nil || len(a.components) != 2 || target.Pointer != "/components/schemas/Root/items/additionalProperties/properties/children/items" || target.Steps != 0 {
		t.Fatal("original node/pointer/catalog", target, err)
	}
	ty, err := a.referenceType(target, nil, 1, "/use")
	if err != nil || ty.Kind != Named || ty.Name != a.internalModels[child] {
		t.Fatal("container nullability leaked into child", ty, err)
	}
	peer := &Model{ClassName: "RootDtoInnerValue", Properties: []Property{{}}, Nested: []Declaration{&Model{ClassName: "ChildrenInner", Properties: []Property{{}}}}}
	if err := a.validateInternalModels([]Declaration{&ValueAlias{ClassName: "RootDto"}, peer}); err != nil {
		t.Fatal("final peer owner", err)
	}
	peer.Nested[0].(*Model).Properties = nil
	if err := a.validateInternalModels([]Declaration{peer}); err == nil || err.Code != "E_REF_UNSUPPORTED" || *err.Pointer != "/use" {
		t.Fatal("syntax invented classifier", err)
	}
}
