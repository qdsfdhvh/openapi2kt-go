package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRawTrueReferenceChainsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/raw-true-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned raw/true note"))
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("raw/true warning denial changed prior output")
	}
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	p.change(t, func(d map[string]any) { schemas(d)["ZRaw"].(map[string]any)["minProperties"] = 1 })
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("counted raw source changed prior output")
	}
}
