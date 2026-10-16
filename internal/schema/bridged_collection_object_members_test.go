package schema

import (
	"strings"
	"testing"
)

func TestBridgedCollectionMemberSourcesAndBudgets(t *testing.T) {
	for _, tc := range []struct {
		reference string
		extra     int
	}{
		{"RowsBridge", 3}, {"RowsHops/properties/first", 2},
	} {
		t.Run(tc.reference, func(t *testing.T) { testCollectionMemberSourcesAndBudgets(t, tc.reference, tc.extra) })
	}
}

func testCollectionMemberSourcesAndBudgets(t *testing.T, reference string, extra int) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Rows":{"type":"array","nullable":true,"items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}},"RowsHops":{"properties":{"first":{"$ref":"#/components/schemas/RowsHops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Rows"}}},"RowsBridge":{"$ref":"#/components/schemas/RowsHops/properties/first"},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/RowsBridge"},"direct":{"$ref":"#/components/schemas/Rows"}}}}}}}}`)
	input = []byte(strings.Replace(string(input), `"member":{"$ref":"#/components/schemas/RowsBridge"}`, `"member":{"$ref":"#/components/schemas/`+reference+`"}`, 1))
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	member, direct := get(properties, "member"), get(properties, "direct")
	rows := get(referencedSchemas(root), "Rows")
	target := get(referencedSchemas(root), "Target")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	if _, err := a.objectDefaultPropertyNode(direct, "/direct", 1, true); err != nil {
		t.Fatal(err)
	}
	directWork := work
	work = 0
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if work != directWork+extra {
		t.Fatal("bridge/hops not charged", directWork, work)
	}
	if node.schema != rows || node.pointer != "/components/schemas/Rows" || node.ty.Kind == Nullable || node.element.element.schema != target || node.element.element.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost original sources/nearest false", node)
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
			t.Fatal("reset shared budget", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-6-extra, true); err != nil {
		t.Fatal("exact source depth rejected", err)
	}
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-5-extra, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost bridge depth", err)
	}
	if a.objectDefaultAllOfMemberElement != nil {
		t.Fatal("permission escaped")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("member role leaked into element")
	}
	hops := a.components["RowsHops"]
	a.hierarchySources[hops.Schema] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy-owned hop admitted")
	}
	delete(a.hierarchySources, hops.Schema)
	alias := a.components["RowsBridge"]
	if extra == 2 {
		alias = a.components["RowsHops"]
	}
	saved := alias.Pointer
	alias.Pointer = "/wrong"
	if _, err = a.referencedCollectionAllOfMemberNode(member, "/member", 1); err == nil {
		t.Fatal("wrong alias identity admitted")
	}
	alias.Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}
