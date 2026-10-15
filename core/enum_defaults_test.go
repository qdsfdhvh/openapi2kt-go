package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumDefaultsDirectDTOFieldsAndRequiredPresence(t *testing.T) {
	input := schemas(`{"Record":{"properties":{"state":{"type":"string","enum":["ready","busy"],"default":"ready"},"count":{"type":"integer","enum":[1,2],"default":2},"required":{"enum":["a","b"],"default":"b"}},"required":["required"]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "RecordDto")
	for _, want := range []string{"val state: State? = `openapi-kt default enum org example models models RecordDto State`(\"ready\"),", `val count: kotlin.Int? = 2,`, `val required: Required,`} {
		if !strings.Contains(text, want) {
			t.Fatal("enum default or required presence lost", want, text)
		}
	}
}
