package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootBorrowedCollectionNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/root-borrowed-collection.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Source"].(map[string]any)["properties"].(map[string]any)["leaf/key~"].(map[string]any)["default"] = "unsupported"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original borrowed source changed previous output")
	}
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/root-borrowed-collection.json")))
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning denial changed old borrowed outputs")
	}
}
