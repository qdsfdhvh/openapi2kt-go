package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestInlineReferencedAllOfFinalSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":7}}}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Inline":{"properties":{"v":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Alias"}},"default":[{"k":{"name":"a"}}]}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	if len(pending) != 1 || has(pending[0].Field.Schema, "$ref") {
		t.Fatal("inline collection not queued through final source barrier", len(pending))
	}
	plan := pending[0]
	if !a.inlineReferencedAllOfCollection(plan.Field.Schema) || !a.deferredFieldDefault(plan.Field.Schema, plan.Field.Pointer) {
		t.Fatal("source role not retained")
	}
	before, _ := root.MarshalJSON()
	node, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if failure != nil {
		t.Fatal(failure)
	}
	source := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
	leaf := node.element.element
	if leaf.schema != source || leaf.pointer != "/components/schemas/Owner/properties/node" || leaf.ty.Name != "org.example.defaults.models.OwnerDto.Node" || leaf.ty.deferredNominal != nil {
		t.Fatal("element lost final physical owner/type or original source", leaf)
	}
	field := get(get(get(source, "allOf").Items()[0], "properties"), "name")
	if leaf.objectFields[0].node.schema != field || leaf.objectFields[0].node.pointer != "/components/schemas/Owner/properties/node/allOf/0/properties/name" {
		t.Fatal("lost original allOf member")
	}
	wrong := TypeRef{Kind: List, Element: &TypeRef{Kind: Map, Element: &TypeRef{Kind: Named, Name: "wrong.Node"}}}
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, wrong, plan.Field.Depth, true); failure == nil {
		t.Fatal("coincident source admitted wrong element classifier")
	}
	a.nominalsLinked = false
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); failure == nil {
		t.Fatal("provisional source admitted inline collection default")
	}
	a.nominalsLinked = true
	binding := a.deferredNominals[source]
	delete(a.deferredNominals, source)
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); failure == nil {
		t.Fatal("missing retained binding admitted inline default")
	}
	a.deferredNominals[source] = binding
	a.hierarchySources[binding.Owner.Schema] = true
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); failure == nil {
		t.Fatal("hierarchy lexical owner admitted inline collection default")
	}
	delete(a.hierarchySources, binding.Owner.Schema)
	// Two container edges, two ref edges, then contribution and property.
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, MaxSchemaDepth-6, true); failure != nil {
		t.Fatal("exact cumulative source48 boundary rejected", failure)
	}
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, MaxSchemaDepth-5, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || *failure.Pointer != "/components/schemas/Owner/properties/node/allOf/0/properties/name" {
		t.Fatal("element-ref path lost source depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) || a.objectDefaultWork != nil {
		t.Fatal("rewrote source or leaked source work")
	}
	original, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	originalBytes, _ := original.MarshalJSON()
	if string(originalBytes) != string(after) {
		t.Fatal("analysis changed original schema/default")
	}
}

func TestInlineReferencedAllOfCandidateAndAnnotationScope(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"n":{"type":"integer"}}}]},"Plain":{"properties":{"name":{"type":"string"}}},"CycleA":{"$ref":"#/components/schemas/CycleB"},"CycleB":{"$ref":"#/components/schemas/CycleA"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := objectCollectionTestAnalyzer(false)
	a.root, a.components = root, map[string]*component{}
	for _, member := range referencedSchemas(root).Members() {
		a.components[member.Name] = &component{Wire: member.Name, Name: member.Name, Schema: member.Value, Pointer: "/components/schemas/" + member.Name}
	}
	for _, text := range []string{
		`{"type":"array","items":{"$ref":"#/components/schemas/Plain"}}`,
		`{"$ref":"#/components/schemas/Item"}`,
		`{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}}}`,
		`{"type":"array","items":{"allOf":[{"properties":{"name":{"type":"string"}}}]}}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/CycleA"}}`,
	} {
		v, err := document.Load([]byte(text), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if a.inlineReferencedAllOfCollection(v) {
			t.Fatal("independent default role gained candidate permission", text)
		}
	}
}
