package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestHierarchyCompoundDispatchNativeAndPreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/hierarchy-dispatch.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check changed output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Range"].(map[string]any)["properties"].(map[string]any)["mode"].(map[string]any)["format"] = "unsupported"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid compound field overwrote output")
	}
}
