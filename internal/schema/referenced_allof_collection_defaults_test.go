package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedAllOfCollectionSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"code":{"type":"integer","enum":[1,2]}}}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Rows":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Alias"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[{"k":{"name":"a","code":2}}]}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	before, _ := root.MarshalJSON()
	plan := pending[0]
	warnings := len(a.warnings)
	node, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if failure != nil {
		t.Fatal(failure)
	}
	leaf := node.element.element
	source := get(get(get(referencedSchemas(root), "Owner"), "properties"), "node")
	if !leaf.object || leaf.schema != source || leaf.pointer != "/components/schemas/Owner/properties/node" || leaf.ty.Name != "org.example.defaults.models.OwnerDto.Node" || len(a.warnings) != warnings {
		t.Fatal("element lost original source/final lexical type or repeated warnings", leaf)
	}
	field := get(get(get(source, "allOf").Items()[0], "properties"), "name")
	if leaf.objectFields[0].node.schema != field || leaf.objectFields[0].node.pointer != "/components/schemas/Owner/properties/node/allOf/0/properties/name" {
		t.Fatal("lost original allOf contribution")
	}
	wrong := TypeRef{Kind: List, Element: &TypeRef{Kind: Map, Element: &TypeRef{Kind: Named, Name: "other.Node"}}}
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, wrong, plan.Field.Depth, true); failure == nil {
		t.Fatal("coincident collection source authorized wrong element classifier")
	}
	binding := a.deferredNominals[source]
	delete(a.deferredNominals, source)
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); failure == nil {
		t.Fatal("missing final source binding authorized element default")
	}
	a.deferredNominals[source] = binding
	a.hierarchySources[binding.Owner.Schema] = true
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); failure == nil {
		t.Fatal("hierarchy lexical owner authorized collection default")
	}
	delete(a.hierarchySources, binding.Owner.Schema)
	// The field/collection/Map/alias/element-ref path reaches the allOf root
	// after5 edges; a contribution and its leaf consume2 more. Exercise the
	// new element-ref call at the actual48 boundary, including an empty value.
	entryDepth := MaxSchemaDepth - 7
	bounded, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, entryDepth, true)
	if failure != nil {
		t.Fatal("exact cumulative source-depth boundary rejected", failure)
	}
	empty, err := document.Load([]byte(`{"default":[]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if encoded, failure := bounded.value(get(empty, "default"), "/default"); failure != nil || encoded != "[]" {
		t.Fatal("bounded empty annotation", encoded, failure)
	}
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, entryDepth+1, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || *failure.Pointer != "/components/schemas/Owner/properties/node/allOf/0/properties/name" {
		t.Fatal("element-reference call lost cumulative source depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) || a.objectDefaultWork != nil {
		t.Fatal("rewrote schema or retained per-walk budget")
	}
	original, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	originalBytes, _ := original.MarshalJSON()
	if string(originalBytes) != string(after) {
		t.Fatal("analysis rewrote source/default values")
	}
}
