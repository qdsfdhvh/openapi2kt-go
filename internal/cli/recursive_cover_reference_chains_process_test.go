package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecursiveCoverReferenceChainsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/recursive-cover-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned cover note"))
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("cover warning denial mutated old output")
	}
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	p.change(t, func(d map[string]any) {
		schemas(d)["ZEmpty"].(map[string]any)["oneOf"].([]any)[1].(map[string]any)["minLength"] = 1
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("bad complete-cover descendant changed old output")
	}
}
