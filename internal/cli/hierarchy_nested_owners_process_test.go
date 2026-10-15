package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestHierarchyNestedOwnersNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/hierarchy-nested-owners.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check mutated output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["First"].(map[string]any)["properties"].(map[string]any)["node"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "invalid"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid nested target overwrote output")
	}
}
