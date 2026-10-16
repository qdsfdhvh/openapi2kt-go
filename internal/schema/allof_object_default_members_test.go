package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestAllOfObjectDefaultMemberSourceDepthAndBudget(t *testing.T) {
	root, err := document.Load([]byte(`{"properties":{"member":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	member := get(get(root, "properties"), "member")
	for _, remaining := range []int{5, 4} {
		a := objectDefaultTestAnalyzer(false)
		work := maxNestedDefaultNodes - remaining
		a.objectDefaultWork = &work
		node, failure := a.objectDefaultNode(root, "/v", 1, true)
		if remaining == 4 {
			if failure == nil || failure.Code != "E_MODEL_LIMIT" || failure.Pointer == nil || *failure.Pointer != "/v/properties/member/allOf/1/properties/b" {
				t.Fatal("member reset source budget", failure)
			}
			continue
		}
		if failure != nil || work != maxNestedDefaultNodes {
			t.Fatal(work, failure)
		}
		child := node.objectFields[0].node
		if child.schema != member || child.pointer != "/v/properties/member" {
			t.Fatal("lost original member", child)
		}
		for i, key := range []string{"a", "b"} {
			original := get(get(get(member, "allOf").Items()[i], "properties"), key)
			if child.objectFields[i].node.schema != original {
				t.Fatal("copied original field", key)
			}
		}
	}
	a := objectDefaultTestAnalyzer(false)
	if _, failure := a.objectDefaultNode(root, "/v", MaxSchemaDepth-1, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost expanded member depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source")
	}
	// Dispatching a property must not grant the same source a collection role.
	if _, failure := a.objectDefaultPropertyNode(member, "/member", 1, true); failure != nil {
		t.Fatal(failure)
	}
	if _, failure := a.objectDefaultChild(member, "/items", 1); failure == nil {
		t.Fatal("member permission leaked into unrelated element walk")
	}
}
