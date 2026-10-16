package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestAliasedAllOfObjectDefaultMemberSourcesAndBudgets(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]},"Alias":{"$ref":"#/components/schemas/Target","nullable":true},"Outer":{"$ref":"#/components/schemas/Alias","nullable":false},"Holder":{"properties":{"v":{"properties":{"direct":{"$ref":"#/components/schemas/Target"},"chain":{"$ref":"#/components/schemas/Outer"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	properties := get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties")
	direct, chain := get(properties, "direct"), get(properties, "chain")
	target := get(referencedSchemas(root), "Target")
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
	if node.schema != target || node.pointer != "/components/schemas/Target" || node.ty.Kind == Nullable {
		t.Fatal("lost original source or nearest false", node)
	}
	for i, key := range []string{"a", "b"} {
		original := get(get(get(target, "allOf").Items()[i], "properties"), key)
		if node.objectFields[i].node.schema != original {
			t.Fatal("copied original field", key)
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
		} else if err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Target/allOf/1/properties/b" {
			t.Fatal("reset budget through aliases", err)
		}
	}
	work = maxNestedDefaultNodes - 1
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/components/schemas/Outer" {
		t.Fatal("alias budget pointer", err)
	}
	a.objectDefaultWork = nil
	// Two aliases consume exactly two additional source-depth edges, even when
	// the catalog has already cached their canonical target.
	for _, tc := range []struct {
		node  *document.Value
		depth int
		valid bool
	}{
		{direct, MaxSchemaDepth - 3, true},
		{chain, MaxSchemaDepth - 5, true},
		{chain, MaxSchemaDepth - 4, false},
	} {
		_, err = a.objectDefaultPropertyNode(tc.node, "/member", tc.depth, true)
		if tc.valid {
			if err != nil {
				t.Fatal("boundary rejected", tc.depth, err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("cached chain lost depth", err)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original nodes")
	}
	if _, err = a.objectDefaultChild(chain, "/items", 1); err == nil {
		t.Fatal("alias member permission leaked to element")
	}
	// Every alias's original identity is required, not only its final target.
	alias := a.componentsByOriginalSchema(get(referencedSchemas(root), "Alias"))
	saved := alias.Pointer
	alias.Pointer = "/components/schemas/Other"
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil {
		t.Fatal("mismatched alias pointer admitted")
	}
	alias.Pointer = saved
	// A bounded source walk also protects direct internal callers from cycles.
	cycle, loadErr := document.Load([]byte(`{"$ref":"#/components/schemas/Alias"}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	alias.Schema = cycle
	if _, err = a.objectDefaultPropertyNode(chain, "/chain", 1, true); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("unbounded alias source cycle", err)
	}
}

func TestAliasedAllOfObjectDefaultMemberRecursiveSource(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"allOf":[{"properties":{"next":{"$ref":"#/components/schemas/Alias"}}},{"properties":{"name":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Target"},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}}}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "v")
	_, err := a.objectDefaultNode(field, "/v", 1, true)
	if err == nil || err.Code != "E_SCHEMA_DEPTH" || err.Pointer == nil || !strings.HasPrefix(*err.Pointer, "/components/schemas/") {
		t.Fatal("unbounded recursive member source", err)
	}
}
