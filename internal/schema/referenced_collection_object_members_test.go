package schema

import "testing"

func TestReferencedCollectionAllOfMemberSourcesAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Rows":{"type":"array","nullable":true,"items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}},"RowsAlias":{"$ref":"#/components/schemas/Rows","nullable":false},"RowsOuter":{"$ref":"#/components/schemas/RowsAlias"},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/RowsOuter"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	rows := get(referencedSchemas(root), "Rows")
	target := get(referencedSchemas(root), "Target")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != rows || node.pointer != "/components/schemas/Rows" || node.ty.Kind == Nullable || node.element.element.schema != target || node.element.element.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost collection/element sources or alias false", node)
	}
	if a.objectDefaultAllOfMemberElement != nil || a.objectDefaultWork != nil {
		t.Fatal("scope/work escaped")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("collection member role leaked into element")
	}
	work := 0
	a.objectDefaultWork = &work
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err != nil {
		t.Fatal(err)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Target/allOf/1/properties/b" {
			t.Fatal("reset collection work", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-8, true); err != nil {
		t.Fatal("exact source depth rejected", err)
	}
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-7, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost reference/container depth", err)
	}
	alias := a.components["RowsAlias"]
	saved := alias.Pointer
	alias.Pointer = "/components/schemas/Wrong"
	if _, err = a.referencedCollectionAllOfMemberNode(member, "/member", 1); err == nil {
		t.Fatal("wrong original alias pointer admitted")
	}
	alias.Pointer = saved
	a.hierarchySources[rows] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy source admitted")
	}
	delete(a.hierarchySources, rows)
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}
