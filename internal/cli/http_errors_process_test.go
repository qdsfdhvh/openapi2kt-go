package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPErrorHelperOwnership(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "errors.json")))
	initial := result(t, p.generate(t), 0)
	var helper string
	for _, file := range initial.Files {
		if strings.HasSuffix(file, "/api/ErrorBody.kt") {
			helper = filepath.Join(p.Output, filepath.FromSlash(file))
		}
	}
	if helper == "" {
		t.Fatal(initial)
	}
	keep := filepath.Join(p.Output, "keep.txt")
	put(t, keep, []byte("keep"))
	filtered := result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	if len(filtered.Files) != 3 {
		t.Fatal(filtered)
	}
	if _, err := os.Stat(helper); !os.IsNotExist(err) {
		t.Fatal("stale helper not removed", err)
	}
	if string(read(t, keep)) != "keep" {
		t.Fatal("unowned file changed")
	}
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
}
