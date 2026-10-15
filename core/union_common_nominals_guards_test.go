package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionCommonNominalsCanonicalAliasesMismatchAndNull(t *testing.T) {
	for _, tc := range []struct{ name, left, right, extra, want string }{
		{"dto-alias", `{"$ref":"#/components/schemas/Alias"}`, `{"$ref":"#/components/schemas/Value"}`, `,"Value":{"properties":{"id":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Value"}`, "ValueDto"},
		{"different-classifiers", `{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Other"}`, `,"Value":{"properties":{"id":{"type":"string"}}},"Other":{"properties":{"id":{"type":"string"}}}`, ""},
		{"nullable-target", `{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value"}`, `,"Value":{"nullable":true,"properties":{"id":{"type":"string"}}}`, "ValueDto?"},
		{"narrow-target", `{"$ref":"#/components/schemas/Value","nullable":false}`, `{"$ref":"#/components/schemas/Value","nullable":false}`, `,"Value":{"nullable":true,"properties":{"id":{"type":"string"}}}`, "ValueDto"},
		{"nullable-allof", `{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value"}`, `,"Value":{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}}}]}`, "ValueDto?"},
		{"hierarchy", `{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Value"}`, `,"Value":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind"}},"First":{"properties":{"id":{"type":"string"}}},"Second":{"properties":{"ok":{"type":"boolean"}}}`, "ValueDto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(tc.left, tc.right, tc.extra, false), "Dto"))
			if err != nil {
				t.Fatal(err)
			}
			text := source(t, got, "ChoiceDto")
			if tc.want == "" {
				if strings.Contains(text, "public abstract val value:") {
					t.Fatal("distinct nominal identities collapsed", text)
				}
			} else if !strings.Contains(text, "public abstract val value: org.example.models.models."+tc.want+"\n") {
				t.Fatal(text)
			}
		})
	}
}

func TestUnionCommonNominalsOriginalSourceAndExcludedOwnership(t *testing.T) {
	// Preserve the original inline source as newly supported variant-only
	// ownership; identical fields do not create a shared nominal declaration.
	original := `{"properties":{"id":{"type":"string"}}}`
	accepted, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(original, original, "", false), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if text := source(t, accepted, "ChoiceDto"); strings.Contains(text, "abstract val value:") || strings.Count(text, "public val value: Value,") != 2 {
		t.Fatal("original inline source acquired shared identity", text)
	}
	for _, tc := range []struct{ field, extra string }{
		{`{"$ref":"#/components/schemas/Missing"}`, ""},
		{`{"$ref":"#/components/schemas/Value","unexpected":true}`, `,"Value":{"properties":{"id":{"type":"string"}}}`},
		{`{"$ref":"#/components/schemas/Value"}`, `,"Value":{"properties":{"id":{"type":"string","format":"unsupported"}}}`},
	} {
		got, err := core.GenerateWithDiagnostics(request(commonDictionaryDocument(tc.field, tc.field, tc.extra, false), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("invalid or scratch nominal permission", tc, got, err)
		}
	}
}
