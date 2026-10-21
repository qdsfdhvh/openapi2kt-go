package schema

import "testing"

func TestStringEnumAliasObjectDefaultMemberAuthorityAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"code":{"type":["string","null"],"enum":[null,"on","null"]}}},"Alias":{"$ref":"#/components/schemas/Source/properties/code","nullable":true},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Alias"}}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Hops/properties/first"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	target := get(get(get(referencedSchemas(root), "Source"), "properties"), "code")
	binding := a.deferredNominals[target]
	bindings := len(a.deferredNominals)
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != target || node.pointer != "/components/schemas/Source/properties/code" || node.enum == nil || node.ty.Kind != Named || work != 4 {
		t.Fatal("lost original vocabulary, false or hop work", node, work)
	}
	if binding == nil || binding.Name != "org.example.defaults.models.SourceDto.Code" || a.deferredNominals[target] != binding || len(a.deferredNominals) != bindings {
		t.Fatal("validation replaced actual nominal authority", binding)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("reset source budget", err)
		}
	}
	a.objectDefaultWork = nil
	for _, depth := range []int{MaxSchemaDepth - 4, MaxSchemaDepth - 3} {
		_, err = a.objectDefaultPropertyNode(member, "/member", depth, true)
		if depth == MaxSchemaDepth-4 {
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
		t.Fatal("mismatched alias export admitted")
	}
	alias.Pointer = saved
	owner := a.components["Source"]
	a.hierarchySources[owner.Schema] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy-owned vocabulary admitted")
	}
	delete(a.hierarchySources, owner.Schema)
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original nodes")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("member permission leaked to enum collection element")
	}
}
