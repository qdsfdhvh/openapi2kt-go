package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedAllOfCollectionObjectDefaultMemberAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node","nullable":true},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second","nullable":false},"second":{"$ref":"#/components/schemas/Alias"}}},"Holder":{"properties":{"v":{"properties":{"member":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Hops/properties/first"}}}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
	ref := get(get(field, "items"), "additionalProperties")
	target := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
	before, _ := root.MarshalJSON()
	if a.inlineReferencedAllOfElement(field) != ref || a.inlineReferencedAllOfElement(ref) != nil {
		t.Fatal("lost original element/root role")
	}
	node, err := a.objectDefaultPropertyNode(field, "/member", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	terminal := node.element.element
	if terminal.schema != target || terminal.pointer != "/components/schemas/Owner/properties/node" || terminal.ty.Kind == Nullable || terminal.objectFields[0].node.schema != get(get(get(target, "allOf").Items()[0], "properties"), "a") {
		t.Fatal("lost source or nearest false", terminal)
	}
	if a.objectDefaultAllOfMemberElement != nil || a.objectDefaultWork != nil {
		t.Fatal("permission/work escaped")
	}
	if _, err = a.objectDefaultChild(ref, "/element", 1); err == nil {
		t.Fatal("permission escaped to other role")
	}
	if _, err = a.objectDefaultNode(field, "/member", 1, true); err == nil {
		t.Fatal("permission escaped to unscoped collection")
	}
	copied, loadErr := document.Load(input, document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	copiedField := get(get(get(get(get(referencedSchemas(copied), "Holder"), "properties"), "v"), "properties"), "member")
	validation := *a
	validation.objectDefaultAllOfMemberElement = get(get(copiedField, "items"), "additionalProperties")
	if _, err = validation.objectDefaultNode(field, "/member", 1, true); err == nil {
		t.Fatal("copied reference granted permission")
	}
	work := 0
	a.objectDefaultWork = &work
	if _, err = a.objectDefaultPropertyNode(field, "/member", 1, true); err != nil {
		t.Fatal(err)
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err = a.objectDefaultPropertyNode(field, "/member", 1, true)
		if spare == steps {
			if err != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Owner/properties/node/allOf/1/properties/b" {
			t.Fatal("reset budget", err)
		}
	}
	a.objectDefaultWork = nil
	if _, err = a.objectDefaultPropertyNode(field, "/member", MaxSchemaDepth-8, true); err != nil {
		t.Fatal("exact source depth rejected", err)
	}
	if _, err = a.objectDefaultPropertyNode(field, "/member", MaxSchemaDepth-7, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost container/alias depth", err)
	}
	binding := a.deferredNominals[target]
	saved := *binding
	for _, mutate := range []func(){
		func() { binding.Pointer = "/wrong" }, func() { binding.Owner = a.components["Hops"] }, func() { binding.Schema = ref }, func() { binding.Name = "" }, func() { delete(a.deferredNominals, target) }, func() { a.hierarchySources[a.components["Owner"].Schema] = true },
	} {
		mutate()
		if _, err = a.objectDefaultPropertyNode(field, "/member", 1, true); err == nil {
			t.Fatal("invalid terminal binding admitted")
		}
		*binding = saved
		a.deferredNominals[target] = binding
		delete(a.hierarchySources, a.components["Owner"].Schema)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source")
	}
}
