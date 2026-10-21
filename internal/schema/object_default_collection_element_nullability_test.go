package schema

import (
	"reflect"
	"testing"
)

func TestObjectDefaultCollectionElementNullableSourceAndBudget(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":["integer","null"],"enum":[null,2,4],"minimum":2},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Code","nullable":false}},"Values":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Code","nullable":true}},"Holder":{"properties":{"v":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"},"values":{"$ref":"#/components/schemas/Values"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	target := get(referencedSchemas(root), "Code")
	checks := a.components["Code"].Validation
	before, _ := root.MarshalJSON()
	for _, tc := range []struct {
		name     string
		nullable bool
	}{{"rows", false}, {"values", true}} {
		member := get(properties, tc.name)
		work := 0
		a.objectDefaultWork = &work
		node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
		if err != nil {
			t.Fatal(err)
		}
		leaf := node.element
		if leaf.schema != target || leaf.pointer != "/components/schemas/Code" || (leaf.ty.Kind == Nullable) != tc.nullable || (node.ty.WithoutRootNull().Element.Kind == Nullable) != tc.nullable || work != 1 {
			t.Fatal("lost original source, edge view or existing work count", tc, leaf, work)
		}
		if !reflect.DeepEqual(leaf.ty.WithoutRootNull().Validation, checks) || a.validationSources[leaf.ty.WithoutRootNull().Validation] != leaf.pointer || a.components["Code"].Validation != checks {
			t.Fatal("lost retained checks/source authority")
		}
		work = maxNestedDefaultNodes - 1
		if _, err := a.objectDefaultPropertyNode(member, "/member", 1, true); err != nil || work != maxNestedDefaultNodes {
			t.Fatal("budget boundary changed", err, work)
		}
		if _, err := a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("source budget reset", err)
		}
		a.objectDefaultWork = nil
		if _, err := a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-1, true); err != nil {
			t.Fatal("existing depth boundary changed", err)
		}
		if _, err := a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("element depth lost", err)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source mutated")
	}
}
