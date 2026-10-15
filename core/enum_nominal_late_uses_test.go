package core_test

import (
	"strings"
	"testing"
)

func TestEnumNominalFirstUnionFieldUse(t *testing.T) {
	input := schemas(`{"Owner":{"properties":{"state":{"enum":["on","off"]}}},"Choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"value":{"$ref":"#/components/schemas/Owner/properties/state"}},"required":["value"]},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}},"required":["label"]}]}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "ChoiceDto"), "value: org.example.models.models.OwnerDto.State") {
		t.Fatal("first union field use did not bind source enum")
	}
}
