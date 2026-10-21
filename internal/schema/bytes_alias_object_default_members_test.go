package schema

import "testing"

func TestBytesAliasObjectDefaultMemberAuthorityAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blob":{"type":["string","null"],"format":"byte"},"Alias":{"$ref":"#/components/schemas/Blob","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Hops/properties/value"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	target := get(referencedSchemas(root), "Blob")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != target || node.pointer != "/components/schemas/Blob" || node.ty.Kind != Scalar || node.ty.Name != "kotlin.ByteArray" || work != 3 {
		t.Fatal("lost original bytes source, false or hop work", node, work)
	}
	for _, spare := range []int{3, 2} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
		if spare == 3 {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("reset source budget", err)
		}
	}
	a.objectDefaultWork = nil
	for _, depth := range []int{MaxSchemaDepth - 3, MaxSchemaDepth - 2} {
		_, err = a.objectDefaultPropertyNode(member, "/member", depth, true)
		if depth == MaxSchemaDepth-3 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("lost hop depth", err)
		}
	}
	alias := a.componentsByOriginalSchema(get(referencedSchemas(root), "Alias"))
	saved := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("mismatched original bytes alias admitted")
	}
	alias.Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("member permission leaked to element")
	}
}
