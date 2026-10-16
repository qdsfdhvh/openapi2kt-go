package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedInlineAllOfOriginalConstructorAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":{"type":"array","items":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer"}}}]}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[{"name":"a","n":2}]}}},"Other":{"properties":{"x":{"type":"string"}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	before, _ := root.MarshalJSON()
	plan := pending[0]
	source := get(get(referencedSchemas(root), "Rows"), "items")
	pointer := "/components/schemas/Rows/items"
	binding := a.deferredNominals[source]
	if binding == nil || binding.Name != "org.example.defaults.models.RowsDtoInner" || binding.Owner != a.components["Rows"] {
		t.Fatal("lost actual root collection peer binding", binding)
	}
	warnings := len(a.warnings)
	node, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if failure != nil {
		t.Fatal(failure)
	}
	leaf := node.element
	if leaf.schema != source || leaf.pointer != pointer || leaf.ty.Name != "RowsDtoInner" || leaf.objectFields[0].node.schema != get(get(get(source, "allOf").Items()[0], "properties"), "name") || len(a.warnings) != warnings {
		t.Fatal("lost original source, relative consuming type or warning ownership", leaf)
	}
	if encoded, err := node.value(get(plan.Field.Schema, "default"), "/default"); err != nil || encoded != `[{"n":2,"name":"a"}]` {
		t.Fatal(encoded, err)
	}
	wrong := TypeRef{Kind: List, Element: &TypeRef{Kind: Named, Name: "other.RowsDtoInner"}}
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, wrong, plan.Field.Depth, true); err == nil {
		t.Fatal("wrong namespace/classifier authorized the original constructor")
	}
	original := *binding
	for _, mutate := range []func(){
		func() { binding.Owner = a.components["Other"] },
		func() { binding.Pointer = "/components/schemas/Other/properties/x" },
		func() { binding.Name = "other.RowsDtoInner" },
	} {
		mutate()
		if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); err == nil {
			t.Fatal("invalid retained source binding authorized the default")
		}
		*binding = original
	}
	delete(a.deferredNominals, source)
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); err == nil {
		t.Fatal("missing actual constructor binding authorized the default")
	}
	a.deferredNominals[source] = binding
	a.nominalsLinked = false
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); err == nil {
		t.Fatal("provisional constructor authorized the default")
	}
	a.nominalsLinked = true
	a.hierarchySources[binding.Owner.Schema] = true
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true); err == nil {
		t.Fatal("hierarchy lexical owner authorized the default")
	}
	delete(a.hierarchySources, binding.Owner.Schema)
	if _, err := a.referencedDefaultSchemaRole(source, pointer, leaf.ty, 2, false, noReferencedDefaultRole); err == nil {
		t.Fatal("collection leaf acquired an unrelated default role")
	}
	// Reference, array, contribution and property edges all count even for [].
	entry := MaxSchemaDepth - 4
	bounded, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, entry, true)
	if failure != nil {
		t.Fatal("exact source-depth boundary rejected", failure)
	}
	empty, err := document.Load([]byte(`{"default":[]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if encoded, failure := bounded.value(get(empty, "default"), "/default"); failure != nil || encoded != "[]" {
		t.Fatal(encoded, failure)
	}
	if _, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, entry+1, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || *failure.Pointer != pointer+"/allOf/0/properties/name" {
		t.Fatal("bare allOf leaf lost cumulative depth", failure)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) || a.objectDefaultWork != nil || a.objectDefaultAllOfElement != nil {
		t.Fatal("source rewritten or per-walk permission/budget leaked")
	}
}
