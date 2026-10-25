package schema

import "testing"

func TestObjectDefaultGraphOriginalSourceAndLinkedAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Text":{"type":"string"},"Values":{"additionalProperties":{"$ref":"#/components/schemas/Text"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values"}},"Alias":{"$ref":"#/components/schemas/Rows"},"Holder":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	uses := []objectDefaultGraphUse{}
	a.objectDefaultGraphUses = &uses
	source := get(get(get(referencedSchemas(root), "Holder"), "properties"), "member")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultPropertyNode(source, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != get(referencedSchemas(root), "Rows") || node.element.schema != get(referencedSchemas(root), "Values") || node.element.element.schema != get(referencedSchemas(root), "Text") {
		t.Fatal("copied or replaced original graph nodes")
	}
	if node.pointer != "/components/schemas/Rows" || node.element.pointer != "/components/schemas/Values" || node.element.element.pointer != "/components/schemas/Text" {
		t.Fatal("lost original graph pointers")
	}
	steps := work
	if steps < 6 || len(uses) < 3 {
		t.Fatal("lost graph hops/work", steps, len(uses))
	}
	if err := a.validateObjectDefaultGraphUses(); err != nil {
		t.Fatal(err)
	}
	values := a.components["Values"]
	saved := values.AliasType
	wrong := TypeRef{Kind: Scalar, Name: "kotlin.Int"}
	values.AliasType = TypeRef{Kind: Map, Element: &wrong}
	if err := a.validateObjectDefaultGraphUses(); err == nil {
		t.Fatal("default graph ignored mutated linked leaf type")
	}
	values.AliasType = saved
	if err := a.validateObjectDefaultGraphUses(); err != nil {
		t.Fatal(err)
	}
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err := a.objectDefaultPropertyNode(source, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal("exact graph budget rejected", work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("graph reset shared work", err)
		}
	}
	a.objectDefaultWork = nil
	alias := a.components["Alias"]
	pointer := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err := a.objectDefaultPropertyNode(source, "/member", 1, true); err == nil {
		t.Fatal("mismatched source identity admitted")
	}
	alias.Pointer = pointer
	if _, err := a.objectDefaultPropertyNode(source, "/member", MaxSchemaDepth, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("source depth reset", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source graph")
	}
}
