package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestRawTupleFactoryOriginalSourceAndFieldPermission(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":false,"default":[null,{"default":{"description":"data","$ref":"literal"}}]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"default":[null]}}}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.rawtuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	c := a.components["Rows"]
	node, err := a.componentDefaultNode(c)
	if err != nil || node.schema != c.Schema || node.element.schema != c.Schema || !node.element.rawJSON || node.element.ty.Kind != JSON {
		t.Fatal("raw factory did not borrow the original projection", err)
	}
	encoded, err := node.value(get(c.Schema, "default"), c.Pointer+"/default")
	if err != nil || encoded != `[null,{"default":{"description":"data","$ref":"literal"}}]` {
		t.Fatal("raw literal metadata-looking data keys were interpreted as schema", encoded, err)
	}
	bytes, _ := c.Schema.MarshalJSON()
	copy, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	shadow := *c
	shadow.Schema = copy
	if _, err := a.componentDefaultNode(&shadow); err == nil {
		t.Fatal("same-byte copy acquired component factory authority")
	}
	field := get(get(a.components["Holder"].Schema, "properties"), "rows")
	if _, err := a.arrayItems(field, "/components/schemas/Holder/properties/rows", 1); err == nil {
		t.Fatal("field own annotation leaked generic array permission")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("raw tuple default changed the original source tree")
	}
}
