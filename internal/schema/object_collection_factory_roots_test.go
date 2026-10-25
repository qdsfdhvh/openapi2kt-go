package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestObjectCollectionFactoryProbeDoesNotGrantRejectedSourcePermission(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","items":{"nullable":true,"oneOf":[{"type":"string"},{"type":"integer"}]},"default":[]}}`)
	source := get(get(get(root, "components"), "schemas"), "Rows")
	a := analyzer{root: root, objectDefaultSchemas: map[*document.Value]bool{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
	c := &component{Schema: source, Pointer: "/components/schemas/Rows", Name: "RowsDto"}
	if a.collectionComponentDefaultCandidate(c) {
		t.Fatal("primitive union gained an ordinary DTO factory role")
	}
	if a.objectDefaultSchemas[source] {
		t.Fatal("rejected probe granted root annotation permission")
	}
}

func TestObjectCollectionFactoryProbeAndOriginalAllOfLeaf(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","items":{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"integer"}}}]},"default":[]}}`)
	source := get(get(get(root, "components"), "schemas"), "Rows")
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, objectDefaultSchemas: map[*document.Value]bool{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
	c := &component{Schema: source, Pointer: "/components/schemas/Rows", Name: "RowsDto"}
	if !a.collectionComponentDefaultCandidate(c) || len(a.objectDefaultSchemas) != 0 || a.objectDefaultWork != nil || a.objectDefaultAllOfMemberElement != nil {
		t.Fatal("successful qualification leaked annotation or leaf permission")
	}
	work := 0
	a.objectDefaultWork = &work
	node, err := a.componentDefaultNode(c)
	if err != nil {
		t.Fatal(err)
	}
	leaf := get(source, "items")
	if node.element.schema != leaf || node.element.objectFields[0].node.schema != get(get(get(leaf, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("component factory replaced original allOf leaf")
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err := a.componentDefaultNode(c)
		if spare == steps && (err != nil || work != maxNestedDefaultNodes) || spare < steps && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("factory reset shared source work", steps, work, err)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) || a.objectDefaultAllOfMemberElement != nil {
		t.Fatal("factory changed source AST or caller permissions")
	}
}
