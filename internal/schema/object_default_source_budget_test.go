package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestObjectDefaultSourceBudgetBoundaryAndIsolation(t *testing.T) {
	root, err := document.Load([]byte(`{"properties":{"left":{"properties":{"s":{"type":"string"}}},"right":{"properties":{"s":{"type":"string"}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	for _, remaining := range []int{7, 6} {
		a := objectDefaultTestAnalyzer(false)
		work := maxNestedDefaultNodes - remaining
		a.objectDefaultWork = &work
		_, failure := a.objectDefaultNode(root, "/field", 1, true)
		if remaining == 7 {
			if failure != nil || work != maxNestedDefaultNodes {
				t.Fatal("exact source budget boundary rejected", failure, work)
			}
		} else if failure == nil || failure.Code != "E_MODEL_LIMIT" || failure.Pointer == nil || *failure.Pointer != "/field/properties/right/properties/s" {
			t.Fatal("nested sibling reset budget or lost original Pointer", failure)
		}
	}
	// Fresh annotations on the same analyzer never inherit a prior walk's work.
	a := objectDefaultTestAnalyzer(false)
	for i := 0; i < 3; i++ {
		if _, failure := a.objectDefaultNode(root, "/field", 1, true); failure != nil || a.objectDefaultWork != nil {
			t.Fatal("source budget escaped the validation copy", failure)
		}
	}
	// allOf/union merged shapes use the same counter for original child nodes.
	work := maxNestedDefaultNodes - 1
	a.objectDefaultWork = &work
	s := &shape{Fields: []field{{Schema: root, Pointer: "/source", Depth: 1}}}
	if _, failure := a.objectDefaultShapeNode(s, "/merged", true); failure == nil || failure.Code != "E_MODEL_LIMIT" || failure.Pointer == nil || *failure.Pointer != "/source" {
		t.Fatal("merged shape restarted source budget", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source budget rewrote original schema")
	}
}
