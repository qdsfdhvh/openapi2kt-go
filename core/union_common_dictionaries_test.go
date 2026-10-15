package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonDictionariesDirectAndNamedBacking(t *testing.T) {
	for _, field := range []string{`{"additionalProperties":{"type":"string"}}`, `{"$ref":"#/components/schemas/Mapping"}`} {
		input := schemas(`{"Mapping":{"additionalProperties":{"type":"string"}},"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":` + field + `},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":` + field + `},"required":["value"]}]}}`)
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.ModelStyle, rq.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, field, err)
				}
				text := source(t, got, "ChoiceDto")
				if !strings.Contains(text, "public abstract val value: kotlin.collections.Map<kotlin.String, kotlin.String>") {
					t.Fatal("common dictionary carrier missing", text)
				}
				if strings.Count(text, "public override val value:") != 3 {
					t.Fatal("known/unknown common implementations missing", text)
				}
			}
		}
	}
}
