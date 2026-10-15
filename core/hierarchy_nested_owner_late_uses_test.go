package core_test

import (
	"strings"
	"testing"
)

func TestHierarchyNestedOwnerLateUsesAndUnionConsumers(t *testing.T) {
	base := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Other":{"properties":{"id":{"type":"string"}}},"Leaf":{"properties":{"node":{"properties":{"id":{"type":"string"}},"required":["id"]},"choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"value":{"$ref":"#/components/schemas/Leaf/properties/node"}},"required":["value"]},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}},"required":["label"]}]}}}`
	r := report(t, schemas(`{`+base+`}`), "Dto")
	if !strings.Contains(source(t, r, "LeafDto"), "public val value: org.example.models.models.LeafDto.Node,") {
		t.Fatal(source(t, r, "LeafDto"))
	}
}

func TestHierarchyNestedOwnerFirstUsedAfterHierarchyPlanning(t *testing.T) {
	input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Other":{"properties":{"id":{"type":"string"}}},"Leaf":{"properties":{"node":{"properties":{"id":{"type":"string"}},"required":["id"]}}},"Consumer":{"properties":{"node":{"$ref":"#/components/schemas/Leaf/properties/node"}},"required":["node"]}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "ConsumerDto"), "public val node: org.example.models.models.LeafDto.Node,") {
		t.Fatal(source(t, r, "ConsumerDto"))
	}
}
