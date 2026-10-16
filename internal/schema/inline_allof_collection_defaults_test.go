package schema

import (
	"reflect"
	"testing"

	"openapi-kt/internal/document"
)

func TestInlineAllOfCollectionOriginalLeafAuthority(t *testing.T) {
	input := []byte(`{"type":"array","items":{"type":"object","additionalProperties":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":7}}}]}},"default":[{"z":{"name":"z"},"a":{"name":"a"}}]}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	leaf := get(get(root, "items"), "additionalProperties")
	if inlineAllOfDefaultElement(root) != leaf || inlineAllOfDefaultElement(leaf) != nil {
		t.Fatal("source-role detection changed the original leaf or admitted root allOf")
	}
	primitive, err := document.Load([]byte(`{"type":"array","items":{"allOf":[{"type":"integer","enum":[1]}]},"default":[]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if inlineAllOfDefaultElement(primitive) != nil {
		t.Fatal("primitive allOf lost its original default/diagnostic route")
	}
	a := objectCollectionTestAnalyzer(true)
	if !a.objectDefaultCandidate(root) {
		t.Fatal("direct inline collection candidate lost")
	}
	if _, failure := a.objectDefaultNode(root, "/field", 1, true); failure == nil {
		t.Fatal("unscoped source walk admitted allOf leaf")
	}
	validation := *a
	validation.objectDefaultAllOfElement = leaf
	node, failure := validation.objectDefaultNode(root, "/field", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	element := node.element.element
	name := get(get(get(leaf, "allOf").Items()[0], "properties"), "name")
	if !element.object || element.schema != leaf || element.objectFields[0].node.schema != name || element.objectFields[0].node.pointer != "/field/items/additionalProperties/allOf/0/properties/name" {
		t.Fatal("merged source fields lost original node/Pointer", element)
	}
	encoded, failure := node.value(get(root, "default"), "/field/default")
	if failure != nil || encoded != `[{"a":{"name":"a"},"z":{"name":"z"}}]` {
		t.Fatal(encoded, failure)
	}
	// A structurally equal copy cannot authorize the actual original leaf.
	copyRoot, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	validation.objectDefaultAllOfElement = inlineAllOfDefaultElement(copyRoot)
	if _, failure := validation.objectDefaultNode(root, "/field", 1, true); failure == nil {
		t.Fatal("coincident source copy authorized the original leaf")
	}
	// The initializer creates its own permission and leaves the caller untouched.
	initializer, failure := a.objectDefaultInitializer(field{Schema: root, Pointer: "/field", Depth: 1})
	if failure != nil || initializer == "" || a.objectDefaultAllOfElement != nil || a.objectDefaultWork != nil {
		t.Fatal("source permission or work escaped the validation copy", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source/default values")
	}
	// The new branch is reached at the cumulative source-depth boundary.
	validation.objectDefaultAllOfElement = leaf
	if _, failure := validation.objectDefaultNode(root, "/field", MaxSchemaDepth-4, true); failure != nil {
		t.Fatal("exact source48 boundary rejected", failure)
	}
	if _, failure := validation.objectDefaultNode(root, "/field", MaxSchemaDepth-3, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || *failure.Pointer != "/field/items/additionalProperties/allOf/0/properties/name" {
		t.Fatal("allOf element traversal lost source depth", failure)
	}
}

func TestInlineAllOfCollectionUsesSharedConstructorPlan(t *testing.T) {
	annotated, err := document.Load([]byte(`{"type":"array","items":{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"n":{"type":"integer"}}}]},"default":[]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := document.Load([]byte(`{"type":"array","items":{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"n":{"type":"integer"}}}]}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectCollectionTestAnalyzer(false)
	f := field{Schema: annotated, Pointer: "/field", Depth: 1}
	var nested, baseline []Declaration
	ty, failure := a.objectDefaultField(f, true, "Value", &nested, map[string]bool{})
	if failure != nil {
		t.Fatal(failure)
	}
	b := objectCollectionTestAnalyzer(false)
	want, _, failure := b.modelPropertyType(plain, "Value", "/field", &baseline, map[string]bool{}, 1)
	if failure != nil || !reflect.DeepEqual(ty, want) || !reflect.DeepEqual(nested, baseline) || len(nested) != 1 {
		t.Fatal("default changed actual nested constructor/type plan", failure)
	}
	if _, failure := a.objectDefaultField(f, false, "Value", nil, nil); failure == nil {
		t.Fatal("nonordinary consumer acquired inline collection default")
	}
	if a.objectDefaultAllOfElement != nil {
		t.Fatal("type planner retained source-walk permission")
	}
}
