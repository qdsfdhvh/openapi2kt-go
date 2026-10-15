package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionInlineValuesKeepOriginalVariantOwners(t *testing.T) {
	object := `{"properties":{"id":{"type":"string"}},"required":["id"]}`
	for _, field := range []string{
		object,
		`{"type":"array","items":` + object + `}`,
		`{"additionalProperties":` + object + `}`,
		`{"type":"array","items":{"additionalProperties":` + object + `}}`,
		`{"allOf":[` + object + `]}`,
		`{"oneOf":[{"properties":{"tag":{"const":"id"},"id":{"type":"string"}},"required":["id"]},{"properties":{"tag":{"const":"ok"},"ok":{"type":"boolean"}},"required":["ok"]}]}`,
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(commonDictionaryDocument(field, field, "", false), "Dto")
				rq.Options.ModelStyle, rq.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, field, err)
				}
				text := source(t, got, "ChoiceDto")
				if strings.Contains(text, "abstract val value:") || strings.Contains(text, "override val value:") {
					t.Fatal("inline values acquired a fabricated common classifier", text)
				}
				if strings.Count(text, "public val value:") != 2 || !strings.Contains(text, "public abstract val kind:") {
					t.Fatal("variant fields or existing scalar common property lost", text)
				}
				if strings.Count(text, "class Value") != 2 {
					t.Fatal("actual inline owners missing", text)
				}
			}
		}
	}
}

func TestUnionInlineValuesDifferentSourcesAndPresence(t *testing.T) {
	left := `{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]}`
	right := `{"properties":{"ok":{"type":"boolean"}},"required":["ok"]}`
	got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(left, right, "", true), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "ChoiceDto")
	if strings.Count(text, "public val value: Value? = null,") != 2 || strings.Contains(text, "abstract val value:") {
		t.Fatal("variant-specific nullable/default intent lost", text)
	}
	if !strings.Contains(text, "public val id: kotlin.String,") || !strings.Contains(text, "public val ok: kotlin.Boolean,") {
		t.Fatal("different original schemas merged", text)
	}
}

func TestUnionInlineValuesStillValidateEveryOriginalField(t *testing.T) {
	valid := `{"properties":{"id":{"type":"string"}}}`
	for _, invalid := range []string{
		`{"properties":{"id":{"type":"string","format":"unsupported"}}}`,
		`{"type":"array","items":{"properties":{"id":{"type":"string","format":"unsupported"}}}}`,
		`{"additionalProperties":{"properties":{"id":{"type":"string","format":"unsupported"}}}}`,
		`{"properties":{"id":{"type":"string"}},"required":["missing"]}`,
		`{"allOf":[{"properties":{"id":{"type":"string","format":"unsupported"}}},{"properties":{"id":{"type":"string"}}}]}`,
		`{"oneOf":[{"properties":{"id":{"type":"string"}}},{"type":"boolean","unsupported":true}]}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(invalid, valid, "", false), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("source validation was skipped", invalid, got, err)
		}
	}
}
