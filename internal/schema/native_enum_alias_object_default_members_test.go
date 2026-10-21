package schema

import (
	"reflect"
	"testing"
)

func TestNativeEnumAliasObjectDefaultMemberAuthorityAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"code":{"type":["integer","null"],"enum":[null,1,2,4],"minimum":2,"multipleOf":2}}},"Alias":{"$ref":"#/components/schemas/Source/properties/code","nullable":true},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Alias"}}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Hops/properties/first"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	target := get(get(get(referencedSchemas(root), "Source"), "properties"), "code")
	origin := a.components["Alias"].InternalModelTarget.BorrowedScalar
	checks := origin.Checks
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != target || node.pointer != "/components/schemas/Source/properties/code" || node.native == nil || node.enum != nil || node.ty.Kind != Scalar || node.ty.Name != "kotlin.Int" || work != 4 {
		t.Fatal("lost original native vocabulary, nullable false or source hops", node, work)
	}
	if !reflect.DeepEqual(node.ty.Validation, checks) || a.validationSources[node.ty.Validation] != node.pointer || origin.Checks != checks || origin.Schema != target {
		t.Fatal("replaced actual source/check authority", node.ty.Validation, checks)
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
			t.Fatal("source budget reset", err)
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
			t.Fatal("hop depth lost", err)
		}
	}
	alias := a.components["Alias"]
	saved := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("mismatched original alias admitted")
	}
	alias.Pointer = saved
	a.hierarchySources[a.components["Source"].Schema] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy-owned native enum admitted")
	}
	delete(a.hierarchySources, a.components["Source"].Schema)
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source rewritten")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("member permission leaked to collection element")
	}
}
