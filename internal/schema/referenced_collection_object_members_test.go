package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedCollectionAllOfMemberSourcesAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Rows":{"type":"array","nullable":true,"items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}},"RowsAlias":{"$ref":"#/components/schemas/Rows","nullable":false},"RowsOuter":{"$ref":"#/components/schemas/RowsAlias"},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/RowsOuter"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	rows := get(referencedSchemas(root), "Rows")
	target := get(referencedSchemas(root), "Target")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != rows || node.pointer != "/components/schemas/Rows" || node.ty.Kind == Nullable || node.element.element.schema != target || node.element.element.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost collection/element sources or alias false", node)
	}
	if a.objectDefaultAllOfMemberElement != nil || a.objectDefaultWork != nil {
		t.Fatal("scope/work escaped")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("collection member role leaked into element")
	}
	work := 0
	a.objectDefaultWork = &work
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err != nil {
		t.Fatal(err)
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
			t.Fatal("reset collection work", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-8, true); err != nil {
		t.Fatal("exact source depth rejected", err)
	}
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-7, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost reference/container depth", err)
	}
	alias := a.components["RowsAlias"]
	saved := alias.Pointer
	alias.Pointer = "/components/schemas/Wrong"
	if _, err = a.referencedCollectionAllOfMemberNode(member, "/member", 1); err == nil {
		t.Fatal("wrong original alias pointer admitted")
	}
	alias.Pointer = saved
	a.hierarchySources[rows] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy source admitted")
	}
	delete(a.hierarchySources, rows)
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}

// A referenced collection with a bare inline ordinary allOf element leaf shares
// the retained constructor role with the consuming-field default. The detector
// returns the original element node; the value walk keeps its merged fields.
func TestReferencedCollectionBareAllOfMemberSourcesAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":{"type":"array","nullable":true,"items":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}},"RowsAlias":{"$ref":"#/components/schemas/Rows","nullable":false},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/RowsAlias"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	element := get(get(referencedSchemas(root), "Rows"), "items")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != get(referencedSchemas(root), "Rows") || node.element == nil || node.element.schema != element || !node.element.object || node.element.objectFields[0].node.schema != get(get(get(element, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost bare leaf sources or alias false", node)
	}
	if a.objectDefaultAllOfMemberElement != nil || a.objectDefaultWork != nil {
		t.Fatal("scope/work escaped")
	}
	if _, err = a.objectDefaultChild(member, "/items", 1); err == nil {
		t.Fatal("collection member role leaked into element")
	}
	work := 0
	a.objectDefaultWork = &work
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err != nil {
		t.Fatal(err)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(member, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Rows/items/allOf/1/properties/b" {
			t.Fatal("reset bare-leaf work", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-5, true); err != nil {
		t.Fatal("exact source depth rejected", err)
	}
	if _, err = a.objectDefaultPropertyNode(member, "/member", MaxSchemaDepth-4, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost bare-leaf depth", err)
	}
	a.hierarchySources[element] = true
	if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
		t.Fatal("hierarchy source admitted")
	}
	delete(a.hierarchySources, element)
	// A structurally equal sibling node cannot borrow the leaf permission.
	cloneBytes, _ := get(get(referencedSchemas(root), "Rows"), "items").MarshalJSON()
	clone, loadErr := document.Load(cloneBytes, document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if a.namedAllOfCollectionMemberElement(clone) != nil {
		t.Fatal("copied leaf borrowed permission")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}

// An original internal allOf element terminal is bound through the shared
// resolver seam so the eager member walk observes the same node/Pointer/
// SourceCatalog its deferred source check requires. A copied node still fails.
func TestReferencedCollectionInternalAllOfMemberSources(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/node"}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Rows"}},"default":{"member":[{"a":"x"}]}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	node := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
	before, _ := root.MarshalJSON()
	n, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	terminal := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
	if n.schema != get(referencedSchemas(root), "Rows") || n.element == nil || n.element.schema != terminal || !n.element.object ||
		n.element.pointer != "/components/schemas/Owner/properties/node" ||
		n.element.objectFields[0].node.schema != get(get(get(terminal, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost collection/element source", n)
	}
	if a.deferredNominals[node] == nil {
		t.Fatal("internal terminal binding not established")
	}
	pointer := "/components/schemas/Owner/properties/node"
	if err := a.internalAllOfDefaultMemberSource(node, pointer, "#"+pointer); err != nil {
		t.Fatal("bound internal terminal rejected", err)
	}
	cloneBytes, _ := node.MarshalJSON()
	clone, loadErr := document.Load(cloneBytes, document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if err := a.internalAllOfDefaultMemberSource(clone, pointer, "#"+pointer); err == nil {
		t.Fatal("copied internal terminal admitted")
	}
	if err := a.internalAllOfDefaultMemberSource(node, pointer, "#/components/schemas/Owner/properties/other"); err == nil {
		t.Fatal("wrong pointer admitted")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}

// An object-default property may reference a collection whose element resolves
// to a plain named ordinary DTO. The walker follows finite transparent alias
// chains and keeps the original terminal object's merged fields and Pointers.
func TestReferencedCollectionPlainDTOElementSources(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7}}},"ItemAlias":{"$ref":"#/components/schemas/Item"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ItemAlias"}},"Holder":{"properties":{"v":{"properties":{"m":{"$ref":"#/components/schemas/Rows"}},"default":{"m":[{"name":"a"}]}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "m")
	item := get(referencedSchemas(root), "Item")
	name := get(get(item, "properties"), "name")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.schema != get(referencedSchemas(root), "Rows") || node.element == nil || !node.element.object ||
		node.element.schema != item || node.element.pointer != "/components/schemas/Item" ||
		node.element.objectFields[0].node.schema != name {
		t.Fatal("lost plain DTO element source/alias terminal", node)
	}
	if a.objectDefaultAllOfMemberElement != nil {
		t.Fatal("member element permission escaped")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}

// A plain-DTO-element collection reached through a named alias or internal
// bridge keeps every original hop while borrowing the terminal plain object.
func TestReferencedCollectionPlainDTOAliasSources(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"RowsAlias":{"$ref":"#/components/schemas/Rows"},"Holder":{"properties":{"v":{"properties":{"m":{"$ref":"#/components/schemas/RowsAlias"}},"default":{"m":[{"name":"a"}]}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	holder := get(referencedSchemas(root), "Holder")
	member := get(get(get(get(holder, "properties"), "v"), "properties"), "m")
	rowsAlias := get(referencedSchemas(root), "RowsAlias")
	rows := get(referencedSchemas(root), "Rows")
	item := get(referencedSchemas(root), "Item")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.object || node.element == nil || !node.element.object || node.schema != rows || node.pointer != "/components/schemas/Rows" {
		t.Fatal("alias did not borrow the terminal collection", node)
	}
	if node.element.schema != item {
		t.Fatal("lost plain DTO element terminal", node)
	}
	if a.components["RowsAlias"] != a.componentsByOriginalSchema(rowsAlias) || a.componentsByOriginalSchema(rowsAlias) == nil {
		t.Fatal("alias source identity changed")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}

// A nested container inside a plain-DTO collection member keeps the same member
// profile and preserves the original container chain and leaf object node.
func TestReferencedCollectionPlainDTONestedContainerSources(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}},"Nested":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}},"Holder":{"properties":{"v":{"properties":{"m":{"$ref":"#/components/schemas/Nested"}},"default":{"m":[[{"name":"a"}]]}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	holder := get(referencedSchemas(root), "Holder")
	member := get(get(get(get(holder, "properties"), "v"), "properties"), "m")
	nested := get(referencedSchemas(root), "Nested")
	item := get(referencedSchemas(root), "Item")
	before, _ := root.MarshalJSON()
	node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.object || node.element == nil || node.schema != nested || node.pointer != "/components/schemas/Nested" {
		t.Fatal("lost nested container source", node)
	}
	inner := node.element
	if inner.element == nil || !inner.element.object || inner.element.schema != item || inner.element.pointer != "/components/schemas/Item" {
		t.Fatal("lost nested plain DTO leaf", inner)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original sources")
	}
}
