package core_test

import (
	"strings"
	"testing"
)

func TestValueClassInternalChainsFirstUnionUse(t *testing.T) {
	input := schemas(`{"Value":{"type":"integer","format":"int64","x-kotlin-value-class":true},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"value":{"$ref":"#/components/schemas/Owner/properties/hop"}},"required":["value"]},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}},"required":["label"]}]}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "ChoiceDto"), "value: org.example.models.models.ValueDto") {
		t.Fatal("first union use lost canonical wrapper")
	}
}
