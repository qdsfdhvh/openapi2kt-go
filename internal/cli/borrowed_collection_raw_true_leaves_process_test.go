package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBorrowedCollectionRawTrueLeavesNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/borrowed-collection-raw-true-leaves.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("source warning denial changed old output")
	}
	// A raw JSON carrier now accepts any default, so this stays a rejection test
	// only through a shape the carrier still does not admit.
	p.change(t, func(d map[string]any) {
		schemas(d)["MaybeText"].(map[string]any)["oneOf"] = []any{
			map[string]any{"type": "string"},
			map[string]any{"type": "integer"},
		}
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original nominal source changed old output")
	}
}
