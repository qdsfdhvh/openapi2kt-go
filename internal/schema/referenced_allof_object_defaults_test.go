package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedAllOfDefaultSourceAndNominalAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"}}}]},"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":{"n":2,"label":"x"}}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	if len(pending) != 1 {
		t.Fatal("unexpected default plans", len(pending))
	}
	plan := pending[0]
	node, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if failure != nil {
		t.Fatal(failure)
	}
	target := get(referencedSchemas(root), "Value")
	if node.schema != target || !node.object || node.ty.Name != "org.example.defaults.models.ValueDto" || len(node.objectFields) != 2 {
		t.Fatal("actual original Model/source not retained", node)
	}
	members := get(target, "allOf").Items()
	if node.objectFields[0].node.schema != get(get(members[0], "properties"), "label") || node.objectFields[0].node.pointer != "/components/schemas/Value/allOf/0/properties/label" || node.objectFields[1].node.schema != get(get(members[1], "properties"), "n") {
		t.Fatal("lost original member node or pointer")
	}
	canonical, failure := node.value(get(plan.Field.Schema, "default"), "/field/default")
	if failure != nil || canonical != `{"label":"x","n":2}` {
		t.Fatal(canonical, failure)
	}
	// A Named spelling alone cannot authorize a different retained classifier.
	if _, failure := a.referencedAllOfObjectDefaultNode(target, "/components/schemas/Value", TypeRef{Kind: Named, Name: "other.ValueDto"}, 2); failure == nil {
		t.Fatal("coincident source shape authorized another classifier")
	}
	a.nominalsLinked = false
	if _, failure := a.referencedAllOfObjectDefaultNode(target, "/components/schemas/Value", node.ty, 2); failure == nil {
		t.Fatal("provisional nominal source authorized a default")
	}
	a.nominalsLinked = true
	binding := a.deferredNominals[target]
	delete(a.deferredNominals, target)
	if _, failure := a.referencedAllOfObjectDefaultNode(target, "/components/schemas/Value", node.ty, 2); failure == nil {
		t.Fatal("missing actual source binding authorized a default")
	}
	a.deferredNominals[target] = binding
	a.hierarchySources[target] = true
	if _, failure := a.referencedAllOfObjectDefaultNode(target, "/components/schemas/Value", node.ty, 2); failure == nil {
		t.Fatal("hierarchy source acquired ordinary allOf defaults")
	}
	delete(a.hierarchySources, target)
	// Consumer narrowing remains independent from physical source nullability.
	nullable, failure := a.referencedAllOfObjectDefaultNode(target, "/components/schemas/Value", node.ty.Null(), 2)
	if failure != nil || nullable.ty.Kind != Nullable {
		t.Fatal(nullable, failure)
	}
	original, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := original.MarshalJSON()
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original schema or default")
	}
}
