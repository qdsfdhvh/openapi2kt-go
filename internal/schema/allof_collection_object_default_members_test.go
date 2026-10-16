package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestAllOfCollectionObjectDefaultMemberSourcesAndScope(t *testing.T) {
	input := []byte(`{"type":"array","items":{"type":"object","additionalProperties":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}] }},"default":[{"z":{"a":"z"},"a":{"a":"a"}}]}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	leaf := get(get(root, "items"), "additionalProperties")
	a := objectCollectionTestAnalyzer(true)
	node, failure := a.objectDefaultPropertyNode(root, "/member", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	element := node.element.element
	if element.schema != leaf || element.pointer != "/member/items/additionalProperties" || element.objectFields[0].node.schema != get(get(get(leaf, "allOf").Items()[0], "properties"), "a") || element.objectFields[0].node.pointer != "/member/items/additionalProperties/allOf/0/properties/a" {
		t.Fatal("lost source identity", element)
	}
	encoded, failure := node.value(get(root, "default"), "/default")
	if failure != nil || encoded != `[{"a":{"a":"a"},"z":{"a":"z"}}]` {
		t.Fatal(encoded, failure)
	}
	if a.objectDefaultAllOfMemberElement != nil || a.objectDefaultAllOfElement != nil || a.objectDefaultWork != nil {
		t.Fatal("permission/work leaked from property copy")
	}
	if _, failure = a.objectDefaultNode(root, "/member", 1, true); failure == nil {
		t.Fatal("permission escaped to unscoped collection")
	}
	if _, failure = a.objectDefaultChild(leaf, "/element", 1); failure == nil {
		t.Fatal("permission escaped to direct child")
	}
	copyRoot, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	scoped := *a
	scoped.objectDefaultAllOfMemberElement = inlineAllOfDefaultElement(copyRoot)
	if _, failure = scoped.objectDefaultNode(root, "/member", 1, true); failure == nil {
		t.Fatal("copy authorized original leaf")
	}
	scoped.objectDefaultAllOfMemberElement = leaf
	if _, failure = scoped.objectDefaultNode(root, "/member", 1, true); failure != nil {
		t.Fatal(failure)
	}
	// Two sibling properties independently authorize their own original leaves;
	// the caller's unrelated permission remains untouched.
	object, err := document.Load([]byte(`{"properties":{"left":`+string(input)+`,"right":`+string(input)+`}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, failure = scoped.objectDefaultNode(object, "/object", 1, true); failure != nil || scoped.objectDefaultAllOfMemberElement != leaf {
		t.Fatal("sibling permission drift", failure)
	}
	work := 0
	a.objectDefaultWork = &work
	if _, failure = a.objectDefaultPropertyNode(root, "/member", 1, true); failure != nil {
		t.Fatal(failure)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, failure = a.objectDefaultPropertyNode(root, "/member", 1, true)
		if spare == steps {
			if failure != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, failure)
			}
		} else if failure == nil || failure.Code != "E_MODEL_LIMIT" || failure.Pointer == nil || *failure.Pointer != "/member/items/additionalProperties/allOf/1/properties/b" {
			t.Fatal("reset shared budget", failure)
		}
	}
	a.objectDefaultWork = nil
	if _, failure = a.objectDefaultPropertyNode(root, "/member", MaxSchemaDepth-4, true); failure != nil {
		t.Fatal("exact depth rejected", failure)
	}
	if _, failure = a.objectDefaultPropertyNode(root, "/member", MaxSchemaDepth-3, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || *failure.Pointer != "/member/items/additionalProperties/allOf/0/properties/a" {
		t.Fatal("lost depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source")
	}
}
