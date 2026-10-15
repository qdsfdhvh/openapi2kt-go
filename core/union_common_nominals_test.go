package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonNominalReferencesCanonicalClassifiers(t *testing.T) {
	for _, body := range []string{`{"properties":{"id":{"type":"string"}},"required":["id"]}`, `{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]}`, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`, `{"oneOf":[{"properties":{"kind":{"const":"first"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"second"},"ok":{"type":"boolean"}},"required":["ok"]}]}`} {
		field := `{"$ref":"#/components/schemas/Value"}`
		input := commonDictionaryDocument(field, field, `,"Value":`+body, false)
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.ModelStyle = style
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, body, err)
				}
				if !strings.Contains(source(t, got, "ChoiceDto"), "public abstract val value: org.example.models.models.ValueDto") {
					t.Fatal("canonical common classifier", source(t, got, "ChoiceDto"))
				}
			}
		}
	}
}
