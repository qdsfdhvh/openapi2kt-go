package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPSpecialScalarTypes(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "special-scalars.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	model := source(t, r, "RecordDto")
	for _, want := range []string{"kotlin.time.Instant", "kotlinx.datetime.LocalDate", "kotlin.ByteArray", "kotlin.Long", "kotlinx.serialization.json.JsonNull", "LongAsStringSerializer"} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q: %s", want, model)
		}
	}
	inputModel := source(t, r, "SpecialInputDto")
	if !strings.Contains(inputModel, "Patch<") || !strings.Contains(inputModel, "LongAsStringSerializer") {
		t.Fatal("bigint Patch must serialize its Long element, not the carrier", inputModel)
	}
	_ = source(t, r, "LongAsStringSerializer")
}
