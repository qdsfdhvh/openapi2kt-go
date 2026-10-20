package schema

import "testing"

func TestScalarAliasObjectDefaultMemberAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"id":{"type":"string","format":"uuid","nullable":true,"minLength":2}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/id"},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Alias","nullable":true}}},"Holder":{"properties":{"v":{"properties":{"direct":{"$ref":"#/components/schemas/Owner/properties/id"},"chain":{"$ref":"#/components/schemas/Hops/properties/first"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	direct, chain := get(properties, "direct"), get(properties, "chain")
	target := get(get(get(referencedSchemas(root), "Owner"), "properties"), "id")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	if _, err := a.objectDefaultPropertyNode(direct, "/direct", 1, true); err != nil {
		t.Fatal(err)
	}
	directWork := work
	work = 0
	node, err := a.objectDefaultPropertyNode(chain, "/chain", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != target || node.pointer != "/components/schemas/Owner/properties/id" || node.ty.Kind == Nullable || work != directWork+3 {
		t.Fatal("lost source/nullable/hop work", node, work, directWork)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("reset budget", err)
		}
	}
	a.objectDefaultWork = nil
	for _, tc := range []struct {
		depth int
		valid bool
	}{{MaxSchemaDepth - 4, true}, {MaxSchemaDepth - 3, false}} {
		_, err = a.objectDefaultPropertyNode(chain, "/chain", tc.depth, true)
		if tc.valid {
			if err != nil {
				t.Fatal("depth", err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("lost depth", err)
		}
	}
	alias := a.componentsByOriginalSchema(get(referencedSchemas(root), "Alias"))
	saved := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("mismatched original alias admitted")
	}
	alias.Pointer = saved
	owner := a.components["Owner"]
	a.hierarchySources[owner.Schema] = true
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("hierarchy-owned terminal admitted")
	}
	delete(a.hierarchySources, owner.Schema)
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original nodes")
	}
	if _, err = a.objectDefaultChild(chain, "/items", 1); err == nil {
		t.Fatal("member permission leaked to element")
	}
}
