package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestSpecialScalarEnumAndConstContexts(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "special-scalar-values.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for name, tokens := range map[string][]string{
		"DayVocabularyDto": {"public sealed class DayVocabularyDto", "serialName: kotlin.String"},
		"ValueDto":         {"public sealed class Tokens", "public sealed class StampsInner", `kotlinx.datetime.LocalDate.parse("2024-02-29")`, `kotlin.time.Instant.parse("2024-02-29T12:34:56Z")`, "fixedBig: kotlin.Long = 9223372036854775807L"},
		"DefaultApi":       {"input: kotlinx.datetime.LocalDate", "input: kotlin.time.Instant", "input: kotlin.ByteArray"},
	} {
		text := source(t, r, name)
		for _, token := range tokens {
			if !strings.Contains(text, token) {
				t.Fatal(name, "missing", token, text)
			}
		}
	}
	for _, file := range r.Artifacts.Files {
		if strings.Contains(file.Path, "LongAsStringSerializer") {
			t.Fatal("const body exits before bigint field annotation and owns no helper", file.Path)
		}
	}
}
