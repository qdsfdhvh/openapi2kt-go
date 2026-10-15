package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestHierarchyFieldsNativeGenerationAndPreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/hierarchy-fields.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check changed generated files")
	}
	p.change(t, func(d map[string]any) { schemas(d)["MissingFirst"].(map[string]any)["required"] = []any{"id", "kind"} })
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("source error overwrote prior files")
	}
}
