package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableObjectReferenceMembersOverrideAndOverlap(t *testing.T) {
	base := `"Value":{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"False":{"$ref":"#/components/schemas/Maybe","nullable":false}`
	for _, ref := range []string{`{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/False"}`, `{"$ref":"#/components/schemas/Maybe","nullable":false}`} {
		input := schemas(`{` + base + `,"Choice":{"oneOf":[` + ref + `,{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil {
			t.Fatal(ref, err)
		}
		if text := source(t, got, "ChoiceDto"); strings.Contains(text, "NullChoiceDto") != (ref == `{"$ref":"#/components/schemas/Value"}`) {
			t.Fatal("member nearest false intent lost", ref, text)
		}
	}
	input := schemas(`{` + base + `,"Other":{"nullable":true,"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Value"},{"$ref":"#/components/schemas/Other"},{"type":"null"}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, w := range got.Warnings {
		if w.Code == "W_ONEOF_NULL_OVERLAP" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("alias null overlap not retained", got.Warnings)
	}
}

func TestNullableObjectReferenceMembersOriginalGrammarAndIntersections(t *testing.T) {
	for _, source := range []string{
		`{"nullable":true,"properties":{"id":{"type":"string","format":"unsupported"}}}`,
		`{"nullable":true,"properties":{"id":{"type":"string"}},"required":["missing"]}`,
		`{"nullable":true,"properties":{"pin":{"const":"only"}}}`,
		`{"nullable":true,"additionalProperties":{"type":"string"}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+source+`,"Choice":{"oneOf":[{"$ref":"#/components/schemas/Value","nullable":true},{"properties":{"ok":{"type":"boolean"}}}]}}`), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("source role/grammar bypassed", source, got, err)
		}
	}
	input := schemas(`{"Value":{"nullable":true,"properties":{"id":{"type":"string"}}},"Choice":{"allOf":[{"$ref":"#/components/schemas/Value","nullable":false},{"properties":{"ok":{"type":"boolean"}}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("ordinary nullable intersection widened", got, err)
	}
}
