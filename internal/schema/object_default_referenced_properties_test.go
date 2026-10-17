package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestObjectDefaultReferencedPropertyAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"},"n":{"type":"integer","default":7}}},"Mode":{"type":"string","enum":["a","b"]},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Leaf","nullable":true},"mode":{"$ref":"#/components/schemas/Mode"}},"default":{"leaf":{"z":"1"},"mode":"a"}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "v")
	node, failure := a.objectDefaultNode(field, "/components/schemas/Holder/properties/v", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	leaf := get(referencedSchemas(root), "Leaf")
	if !node.object || node.objectFields[0].wire != "leaf" || !node.objectFields[0].node.object || node.objectFields[0].node.schema != leaf {
		t.Fatal("referenced object property did not borrow the target", node.objectFields[0])
	}
	if node.objectFields[0].node.ty.Kind != Nullable {
		t.Fatal("referenced property nullability override lost", node.objectFields[0].node.ty)
	}
	if mode := node.objectFields[1].node; mode == nil || mode.enum == nil || mode.ty.Kind != Named {
		t.Fatal("referenced enum property did not borrow the target vocabulary", mode)
	}
	canonical, failure := node.value(get(field, "default"), "/components/schemas/Holder/properties/v/default")
	if failure != nil || canonical != `{"leaf":{"z":"1"},"mode":"a"}` {
		t.Fatal(canonical, failure)
	}
	// A referenced member value keeps the consuming member Pointer.
	badValue, err := document.Load([]byte(`{"leaf":{}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, failure := node.value(badValue, "/components/schemas/Holder/properties/v/default"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/components/schemas/Holder/properties/v/default/leaf" {
		t.Fatalf("missing required referenced member Pointer: %v", failure)
	}
}

// A finite transparent chain to a plain object terminal keeps every original
// hop, its nearest nullable intent and the shared source-work budget.
func TestObjectDefaultPlainObjectAliasAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"},"n":{"type":"integer","default":7}}},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":true},"Outer":{"$ref":"#/components/schemas/Alias","nullable":false},"Holder":{"properties":{"v":{"properties":{"direct":{"$ref":"#/components/schemas/Leaf"},"chain":{"$ref":"#/components/schemas/Outer"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	direct, chain := get(properties, "direct"), get(properties, "chain")
	leaf := get(referencedSchemas(root), "Leaf")
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
	if work != directWork+2 {
		t.Fatal("alias hops not charged", directWork, work)
	}
	if node.schema != leaf || node.pointer != "/components/schemas/Leaf" || node.ty.Kind == Nullable {
		t.Fatal("lost original terminal or nearest false", node)
	}
	// Every alias's original identity is required, not only its final target.
	alias := a.componentsByOriginalSchema(get(referencedSchemas(root), "Alias"))
	saved := alias.Pointer
	alias.Pointer = "/components/schemas/Other"
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("mismatched alias pointer admitted")
	}
	alias.Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original nodes")
	}
}
