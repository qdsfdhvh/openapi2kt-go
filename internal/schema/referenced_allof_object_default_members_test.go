package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedAllOfObjectDefaultMemberSource(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Target","nullable":true}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "v")
	target := get(referencedSchemas(root), "Target")
	before, _ := root.MarshalJSON()
	work := 0
	a.objectDefaultWork = &work
	node, failure := a.objectDefaultNode(field, "/components/schemas/Holder/properties/v", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	member := node.objectFields[0].node
	if member.schema != target || member.pointer != "/components/schemas/Target" || member.ty.Kind != Nullable {
		t.Fatal("lost target identity or nullable", member)
	}
	for i, key := range []string{"a", "b"} {
		source := get(get(get(target, "allOf").Items()[i], "properties"), key)
		if member.objectFields[i].node.schema != source {
			t.Fatal("copied field", key)
		}
	}
	steps := work
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, failure = a.objectDefaultNode(field, "/v", 1, true)
		if spare == steps {
			if failure != nil || work != maxNestedDefaultNodes {
				t.Fatal(work, failure)
			}
		} else if failure == nil || failure.Code != "E_MODEL_LIMIT" || failure.Pointer == nil || *failure.Pointer != "/components/schemas/Target/allOf/1/properties/b" {
			t.Fatal("reset source budget", failure)
		}
	}
	a.objectDefaultWork = nil
	if _, failure = a.objectDefaultNode(field, "/v", MaxSchemaDepth-2, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("lost ref edge depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original source")
	}
	ref := get(get(field, "properties"), "member")
	// The named member role cannot leak to an element or a different lexical owner.
	if _, failure = a.objectDefaultChild(ref, "/items", 1); failure == nil {
		t.Fatal("member admitted as element")
	}
	owner := a.componentsByOriginalSchema(target)
	oldPointer := owner.Pointer
	owner.Pointer = "/components/schemas/Other"
	if _, failure = a.objectDefaultPropertyReference(ref, "/member", 1); failure == nil {
		t.Fatal("mismatched original pointer admitted")
	}
	owner.Pointer = oldPointer
	a.hierarchySources[target] = true
	if _, failure = a.objectDefaultPropertyReference(ref, "/member", 1); failure == nil {
		t.Fatal("hierarchy admitted")
	}
}

func TestReferencedAllOfObjectDefaultMemberRecursiveSources(t *testing.T) {
	// Finite values and omitted recursive members must not bypass the bounded
	// source walk, even though the ordinary nominal type graph is legal.
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"next":{"$ref":"#/components/schemas/Target"}}},{"properties":{"name":{"type":"string"}}}]},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Target"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "v")
	_, failure := a.objectDefaultNode(field, "/v", 1, true)
	if failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || !strings.HasPrefix(*failure.Pointer, "/components/schemas/Target") {
		t.Fatal(failure)
	}
	// The original loader node remains an object; no expanded schema is installed.
	if get(referencedSchemas(root), "Target").Kind() != document.Object {
		t.Fatal("lost source")
	}
}
