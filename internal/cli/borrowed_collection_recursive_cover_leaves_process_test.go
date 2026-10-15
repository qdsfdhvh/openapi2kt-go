package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBorrowedCollectionRecursiveCoverLeavesNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/borrowed-collection-recursive-cover-leaves.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("source warning denial changed old output")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["Text"].(map[string]any)["oneOf"].([]any)[1].(map[string]any)["minLength"] = 1
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original nominal source changed old output")
	}
}
