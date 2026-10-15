package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJSONUniqueDirectValueArrays(t *testing.T) {
	for _, items := range []string{`true`, `{}`, `{"type":"object"}`, `{"additionalProperties":true}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+items+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(items, err)
		}
		text := source(t, got, "RecordDto")
		if !strings.Contains(text, "`openapi-kt unique JSON items`(value)") || !strings.Contains(text, "private fun `openapi-kt JSON equality key`") {
			t.Fatal("JSON uniqueness lost", text)
		}
	}
}
