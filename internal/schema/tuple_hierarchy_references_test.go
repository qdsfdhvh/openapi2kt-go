package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

const tupleHierarchySources = `"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind","mapping":{"group":"#/components/schemas/Group","other":"#/components/schemas/Other"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"sub","mapping":{"first":"#/components/schemas/First","second":"#/components/schemas/Second"}}},"First":{"properties":{"id":{"type":"string"}},"required":["id"]},"Second":{"properties":{"name":{"type":"string"}},"required":["name"]},"Other":{"properties":{"value":{"type":"string"}},"required":["value"]}`

func TestTupleHierarchySourceAndActualDescendants(t *testing.T) {
	root := loadComponentDocument(t, `{`+tupleHierarchySources+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Root"}],"default":[{"kind":"group","sub":"first","id":"ready"}]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.hierarchytuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	first := a.tupleHierarchyByName("FirstDto")
	group := a.tupleHierarchyByName("GroupDto")
	if first == nil || group == nil {
		t.Fatal("missing actual hierarchy descendants")
	}
	group.SerializeVia = []string{"OtherDto"}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("invented actual serialization route retained tuple authority")
	}
	group.SerializeVia = nil
	parent := first.Parent
	first.Parent = "org.example.hierarchytuples.models.OtherDto"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("reparented actual leaf kept tuple authority")
	}
	first.Parent = parent
	body := first.tagBodies["kind"]
	first.tagBodies["kind"] = "other"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated actual discriminator body retained annotation authority")
	}
	first.tagBodies["kind"] = body
	copyLeaf := *first.Leaf
	leaf := first.Leaf
	first.Leaf = &copyLeaf
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte copied hierarchy constructor acquired actual authority")
	}
	first.Leaf = leaf
	enum := group.Enums[0]
	wire := enum.Members[0].Wire
	enum.Members[0].Wire = "other"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated inherited discriminator vocabulary kept tuple authority")
	}
	enum.Members[0].Wire = wire
	for _, d := range ds {
		if alias, ok := d.(*ValueAlias); ok && alias.Name() == "RowsDto" {
			original := alias.Type.Element
			copyType := *original
			alias.Type.Element = &copyType
			if a.validateTuplePlan(plan) == nil {
				t.Fatal("copied actual hierarchy element acquired tuple use authority")
			}
			alias.Type.Element = original
		}
	}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	mapping := get(get(a.components["Group"].Schema, "discriminator"), "mapping")
	original := mapping.Members()[0].Value
	data, _ := original.MarshalJSON()
	wrapped := append([]byte(`{"value":`), data...)
	wrapped = append(wrapped, '}')
	copyValue, failure := document.Load(wrapped, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	mapping.Members()[0].Value = get(copyValue, "value")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original nested mapping replacement kept source proof")
	}
	mapping.Members()[0].Value = original
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("hierarchy tuple planning mutated source")
	}
}
