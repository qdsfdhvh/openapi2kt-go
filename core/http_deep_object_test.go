package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPDeepObject(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "deep-object.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	model := source(t, r, "SearchRecordsFilterDto")
	for _, want := range []string{"token: kotlin.String? = null", "memo: kotlin.String? = null", "snake_case: kotlin.String? = null"} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q: %s", want, model)
		}
	}
	for _, file := range r.Artifacts.Files {
		if strings.Contains(file.Content, "Patch<") || strings.HasSuffix(file.Path, "/Patch.kt") {
			t.Fatal("query model acquired request-only Patch", file.Path)
		}
	}
	api := source(t, r, "DefaultApi")
	want := "id: kotlin.String, limit: kotlin.Int? = null, filter: org.example.models.api.models.SearchRecordsFilterDto? = null, xTrace: kotlin.String? = null"
	if !strings.Contains(strings.Join(strings.Fields(api), " "), want) {
		t.Fatalf("deepObject positional order: %s", api)
	}
}
