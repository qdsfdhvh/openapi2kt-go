package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCollectionUniquePropagation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/collection-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RecordDto", "UniqueApi", "PatchInputDto", "WatchEventDto"} {
		text := source(t, r, name)
		if strings.Count(text, "private fun <T : kotlin.Comparable<T>>") != 1 || strings.Contains(text, "private inline") {
			t.Fatal("helper not private non-inline and deduplicated", name, text)
		}
		if !strings.Contains(text, "uniqueItems") || !strings.Contains(text, ".sorted()") {
			t.Fatal("lost uniqueness checks", name, text)
		}
	}
	text := source(t, r, "RecordDto")
	for _, check := range []string{"minItems", "maxItems", "minLength", "?.let"} {
		if !strings.Contains(text, check) {
			t.Fatal("lost coexisting check", check, text)
		}
	}
}
