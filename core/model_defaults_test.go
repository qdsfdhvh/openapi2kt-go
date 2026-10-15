package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestModelDefaultsOptionalLiteralAndRequiredPresence(t *testing.T) {
	input := schemas(`{"Record":{"properties":{"label":{"type":"string","default":"hello"},"enabled":{"type":"boolean","default":true},"count":{"type":"integer","default":7},"required":{"type":"string","default":"required"}},"required":["required"]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "RecordDto")
	for _, want := range []string{`val label: kotlin.String? = "hello",`, `val enabled: kotlin.Boolean? = true,`, `val count: kotlin.Int? = 7,`, `val required: kotlin.String,`} {
		if !strings.Contains(text, want) {
			t.Fatal("default or required presence lost", want, text)
		}
	}
}
