package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestFixtureLayoutIndependentOfWorkingDirectory(t *testing.T) {
	repo := testutil.RepoRoot()
	want := filepath.Join(repo, "tests", "fixtures", "p1")
	if got := testutil.Root(); got != want {
		t.Fatalf("fixture root %q, want %q", got, want)
	}
	t.Chdir(t.TempDir())
	if testutil.RepoRoot() != repo || testutil.Root() != want {
		t.Fatal("test paths depend on cwd")
	}
	for _, relative := range []string{"go.mod", "cmd/openapi-kt/main.go", "samples/kotlin-smoke/README.md", "tests/fixtures/catalog.json", "tests/fixtures/model-semantics/dictionaries.json"} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); err != nil {
			t.Fatal(err)
		}
	}
	if len(testutil.Read(t, "cases.json")) == 0 {
		t.Fatal("empty corpus")
	}
	if got := len(testutil.Corpus(t)); got != 154 {
		t.Fatal("corpus inventory changed", got)
	}
}
