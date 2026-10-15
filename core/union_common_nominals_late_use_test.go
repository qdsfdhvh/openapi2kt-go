package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCommonNominalLateInlineHTTPAndInternalCanonicalIdentity(t *testing.T) {
	choice := `{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/node"}},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":{"$ref":"#/components/schemas/Value"}},"required":["value"]}]}`
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]},"Hops":{"properties":{"node":{"$ref":"#/components/schemas/Value"}}}}},"paths":{"/submit":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + choice + `}}},"responses":{"204":{}}}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("first late typed common field", checked, err)
		}
		text := ""
		for _, file := range got.Artifacts.Files {
			text += file.Content
		}
		if !strings.Contains(text, "public abstract val value: org.example.models.models.ValueDto?") {
			t.Fatal("internal/direct canonical common identity lost", text)
		}
	}
}
