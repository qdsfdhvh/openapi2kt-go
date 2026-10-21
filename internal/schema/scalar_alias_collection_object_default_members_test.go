package schema

import (
	"reflect"
	"testing"
)

func TestScalarAliasCollectionObjectDefaultAuthorityAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"code":{"type":["integer","null"],"enum":[null,2,4],"minimum":2},"hop":{"$ref":"#/components/schemas/Owner/properties/code","nullable":true}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop","nullable":false},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Rows"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	target := get(get(get(referencedSchemas(root), "Owner"), "properties"), "code")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	leaf := node.element
	if leaf.schema != target || leaf.pointer != "/components/schemas/Owner/properties/code" || leaf.ty.Kind == Nullable || work != 3 {
		t.Fatal("lost original source, nullable or hop work", leaf, work)
	}
	checks := leaf.ty.Validation
	if checks == nil || a.validationSources[checks] != leaf.pointer || !reflect.DeepEqual(node.ty.WithoutRootNull().Element.Validation, checks) {
		t.Fatal("lost retained checks/source authority")
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal("budget boundary", err, work)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("budget reset", err)
		}
	}
	a.objectDefaultWork = nil
	for _, tc := range []struct {
		depth int
		valid bool
	}{{MaxSchemaDepth - 3, true}, {MaxSchemaDepth - 2, false}} {
		_, err = a.objectDefaultPropertyNode(member, "/member", tc.depth, true)
		if tc.valid {
			if err != nil {
				t.Fatal("depth boundary", err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("lost element depth", err)
		}
	}
	alias := a.components["Alias"]
	saved := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("mismatched original alias admitted")
	}
	alias.Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original source")
	}
}
