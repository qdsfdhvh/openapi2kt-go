package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func nullableObjectMemberDocument(member string) []byte {
	return schemas(`{"Choice":{"oneOf":[` + member + `,{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Choice"}},"required":["value"]}}`)
}

func TestNullableObjectMembersPreservePhysicalVariantsAndNull(t *testing.T) {
	for _, intent := range []string{`"nullable":true,`, `"type":["object","null"],`} {
		member := `{` + intent + `"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}`
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(nullableObjectMemberDocument(member), "Dto")
			rq.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(intent, style, err)
			}
			text := source(t, got, "ChoiceDto")
			for _, part := range []string{"NullChoiceDto", "public val id: kotlin.String,", "if (element == kotlinx.serialization.json.JsonNull)", "public abstract val kind:"} {
				if !strings.Contains(text, part) {
					t.Fatal("nullable member lost original variant/carrier", part, text)
				}
			}
		}
	}
}

func TestNullableObjectMembersInlineUsesNullableCodec(t *testing.T) {
	input := fieldDoc(`{"oneOf":[{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if text := source(t, got, "RecordDto"); !strings.Contains(text, "public val value: Value? = null,") || strings.Contains(text, "NullValue") {
		t.Fatal("inline null intent/carrier mismatch", text)
	}
}
