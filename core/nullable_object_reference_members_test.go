package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableObjectReferenceMembersPhysicalSourcesAndIntent(t *testing.T) {
	for _, intent := range []string{`"nullable":true,`, ""} {
		for _, member := range []string{`{"$ref":"#/components/schemas/Value","nullable":true}`, `{"$ref":"#/components/schemas/Maybe"}`} {
			input := schemas(`{"Value":{` + intent + `"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Choice":{"oneOf":[` + member + `,{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Choice"}},"required":["value"]}}`)
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err != nil {
				t.Fatal(intent, member, err)
			}
			text := source(t, got, "ChoiceDto")
			if !strings.Contains(text, "NullChoiceDto") || !strings.Contains(text, "public val id: kotlin.String,") {
				t.Fatal("member source or null intent lost", text)
			}
		}
	}
}
