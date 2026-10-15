package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecursiveJSONReferenceGraphNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/recursive-json-reference-graph.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check changed old output")
	}
	p.change(t, func(d map[string]any) { schemas(d)["Through"].(map[string]any)["nullable"] = false })
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("graph ref grammar changed old output")
	}
}
