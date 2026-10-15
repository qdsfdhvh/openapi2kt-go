package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBorrowedCollectionProductUnionLeavesNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/borrowed-collection-product-union-leaves.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Node"].(map[string]any)["allOf"].([]any)[0].(map[string]any)["oneOf"].([]any)[0].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["default"] = "unsupported"
	})
	// The original representable default is now supported through this DTO.
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	before = snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Node"].(map[string]any)["allOf"].([]any)[0].(map[string]any)["oneOf"].([]any)[0].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["default"] = true
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original nominal source changed old output")
	}
}
