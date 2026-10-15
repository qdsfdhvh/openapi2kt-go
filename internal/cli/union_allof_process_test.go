package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnionAllOfNativeGeneration(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/union-allof.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check mutated output")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["Common"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
	})
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("source error changed output")
	}
}
