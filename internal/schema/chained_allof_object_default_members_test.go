package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestChainedAllOfObjectDefaultMemberSourcesAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}}},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Owner/properties/node","nullable":true}}},"Holder":{"properties":{"v":{"properties":{"direct":{"$ref":"#/components/schemas/Owner/properties/node"},"chain":{"$ref":"#/components/schemas/Hops/properties/first"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	direct, chain := get(properties, "direct"), get(properties, "chain")
	target := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
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
	if work != directWork+2 || node.ty.Kind == Nullable || node.schema != target || node.pointer != "/components/schemas/Owner/properties/node" {
		t.Fatal("lost hop work, source or nearest false", directWork, work, node)
	}
	for i, key := range []string{"a", "b"} {
		if node.objectFields[i].node.schema != get(get(get(target, "allOf").Items()[i], "properties"), key) {
			t.Fatal("copied source", key)
		}
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Owner/properties/node/allOf/1/properties/b" {
			t.Fatal("reset hop budget", err)
		}
	}
	work = maxNestedDefaultNodes - 1
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Hops/properties/first" {
		t.Fatal("hop limit pointer", err)
	}
	a.objectDefaultWork = nil
	for _, tc := range []struct {
		node  *document.Value
		depth int
		valid bool
	}{
		{direct, MaxSchemaDepth - 3, true}, {chain, MaxSchemaDepth - 5, true}, {chain, MaxSchemaDepth - 4, false},
	} {
		_, err = a.objectDefaultPropertyNode(tc.node, "/member", tc.depth, true)
		if tc.valid {
			if err != nil {
				t.Fatal("valid depth", err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("cached hops lost depth", err)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source")
	}
	if _, err = a.objectDefaultChild(chain, "/items", 1); err == nil {
		t.Fatal("member permission leaked to element")
	}
	hopOwner := a.components["Hops"]
	hop := get(get(hopOwner.Schema, "properties"), "first")
	pointer := "/components/schemas/Hops/properties/first"
	if err := a.internalDefaultMemberSource(hop, pointer, "#"+pointer); err != nil {
		t.Fatal(err)
	}
	cloneBytes, _ := hop.MarshalJSON()
	clone, loadErr := document.Load(cloneBytes, document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if err := a.internalDefaultMemberSource(clone, pointer, "#"+pointer); err == nil {
		t.Fatal("copied hop admitted")
	}
	if err := a.internalDefaultMemberSource(hop, pointer, "#/components/schemas/Hops/properties/second"); err == nil {
		t.Fatal("wrong original hop admitted")
	}
	a.hierarchySources[hopOwner.Schema] = true
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("hierarchy-owned hop admitted")
	}
	delete(a.hierarchySources, hopOwner.Schema)
	// Transparent hops cannot substitute for the terminal's nominal authority.
	binding := a.deferredNominals[target]
	delete(a.deferredNominals, target)
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("missing terminal binding admitted")
	}
	a.deferredNominals[target] = binding
	// The source walker remains bounded independently of the typed resolver.
	cycle, loadErr := document.Load([]byte(`{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/first"}}}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	hopOwner.Schema = cycle
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("unbounded source cycle", err)
	}
}
