package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestInternalAllOfDefaultSourceAndOwnerAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":{"n":2,"label":"x"}}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Owner":{"properties":{"node":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"}}}]}}},"Other":{"properties":{"x":{"type":"string"}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	if len(pending) != 1 {
		t.Fatal("unexpected plans", len(pending))
	}
	plan := pending[0]
	owner := a.components["Owner"]
	target := get(get(owner.Schema, "properties"), "node")
	pointer := "/components/schemas/Owner/properties/node"
	binding := a.deferredNominals[target]
	if binding == nil || binding.Owner != owner || binding.Pointer != pointer || binding.Name != "org.example.defaults.models.OwnerDto.Node" {
		t.Fatal("lost final original lexical binding", binding)
	}
	node, failure := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if failure != nil || node.schema != target || node.ty.Name != binding.Name || len(node.objectFields) != 2 {
		t.Fatal("actual internal source/type", node, failure)
	}
	branches := get(target, "allOf").Items()
	if node.objectFields[0].node.schema != get(get(branches[0], "properties"), "label") || node.objectFields[0].node.pointer != pointer+"/allOf/0/properties/label" {
		t.Fatal("lost original field source/Pointer")
	}
	encoded, failure := node.value(get(plan.Field.Schema, "default"), "/field/default")
	if failure != nil || encoded != `{"label":"x","n":2}` {
		t.Fatal(encoded, failure)
	}
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, TypeRef{Kind: Named, Name: "other.OwnerDto.Node"}, 2); failure == nil {
		t.Fatal("coincident classifier authorized a default")
	}
	originalBinding := *binding
	binding.Owner = a.components["Other"]
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, node.ty, 2); failure == nil {
		t.Fatal("wrong catalog authorized original node")
	}
	*binding = originalBinding
	// A bound name is insufficient if the original schema path has changed.
	binding.Pointer = "/components/schemas/Other/properties/x"
	if _, failure := a.referencedAllOfObjectDefaultNode(target, binding.Pointer, node.ty, 2); failure == nil {
		t.Fatal("wrong original path authorized a default")
	}
	*binding = originalBinding
	a.hierarchySources[owner.Schema] = true
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, node.ty, 2); failure == nil {
		t.Fatal("hierarchy lexical owner acquired ordinary default role")
	}
	delete(a.hierarchySources, owner.Schema)
	a.nominalsLinked = false
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, node.ty, 2); failure == nil {
		t.Fatal("provisional binding authorized internal default")
	}
	a.nominalsLinked = true
	delete(a.deferredNominals, target)
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, node.ty, 2); failure == nil {
		t.Fatal("missing retained model authorized internal default")
	}
	a.deferredNominals[target] = binding
	if _, failure := a.referencedAllOfObjectDefaultNode(target, pointer, node.ty, 48); failure == nil || failure.Code != "E_SCHEMA_DEPTH" || failure.Pointer == nil || !strings.HasPrefix(*failure.Pointer, pointer) {
		t.Fatal("source-depth reset", failure)
	}
	original, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := original.MarshalJSON()
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote original node/default or fabricated catalog")
	}
}
