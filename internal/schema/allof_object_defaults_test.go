package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestOrdinaryAllOfDefaultBorrowsOriginalFields(t *testing.T) {
	root, err := document.Load([]byte(`{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"properties":{"c":{"type":"boolean"}},"default":{"c":true,"b":2,"a":"x"}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original, _ := root.MarshalJSON()
	a := objectDefaultTestAnalyzer(true)
	a.objectDefaultSchemas = map[*document.Value]bool{root: true}
	node, failure := a.ordinaryAllOfDefaultNode(root, "/v", 1)
	if failure != nil {
		t.Fatal(failure)
	}
	if node.schema != root || !node.object || len(node.objectFields) != 3 {
		t.Fatal("lost original object source", node)
	}
	members := get(root, "allOf").Items()
	for i, tc := range []struct {
		value   *document.Value
		pointer string
	}{
		{get(get(members[0], "properties"), "a"), "/v/allOf/0/properties/a"},
		{get(get(members[1], "properties"), "b"), "/v/allOf/1/properties/b"},
		{get(get(root, "properties"), "c"), "/v/properties/c"},
	} {
		if node.objectFields[i].node.schema != tc.value || node.objectFields[i].node.pointer != tc.pointer {
			t.Fatal("merged field lost original source", i)
		}
	}
	encoded, failure := node.value(get(root, "default"), "/v/default")
	if failure != nil || encoded != `{"a":"x","b":2,"c":true}` {
		t.Fatal(encoded, failure)
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(original) {
		t.Fatal("rewrote original allOf schema/default")
	}
}

func TestOrdinaryAllOfWrapperDetectionRetainsSource(t *testing.T) {
	for _, text := range []string{
		`{"allOf":[{"$ref":"#/components/schemas/Value"}],"default":{}}`,
		`{"allOf":[{"description":"annotation"},{"$ref":"#/components/schemas/Value"},{"nullable":true}],"default":{}}`,
	} {
		root, err := document.Load([]byte(text), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if wrapper, failure := referenceWrapper(root, "/v"); failure != nil || wrapper != nil {
			t.Fatal("legacy reference wrapper admitted default", wrapper, failure)
		}
		if wrapper, failure := parseReferenceWrapper(root, "/v", true); failure != nil || wrapper == nil {
			t.Fatal("default wrapper was not detected", wrapper, failure)
		}
		// Detection alone cannot admit the wrapper into the new default role.
		if _, failure := objectDefaultTestAnalyzer(false).ordinaryAllOfDefaultNode(root, "/v", 1); failure == nil || failure.Code != "E_SCHEMA_UNSUPPORTED" || failure.Pointer == nil || *failure.Pointer != "/v/default" {
			t.Fatal("wrapper default changed nominal type", failure)
		}
	}
}
