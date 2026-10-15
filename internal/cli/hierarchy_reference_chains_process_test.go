package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestHierarchyReferenceChainsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/hierarchy-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check mutated output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Hop/key~"].(map[string]any)["properties"].(map[string]any)["parentAlias"].(map[string]any)["$ref"] = "#/components/schemas/Missing"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_REF_MISSING")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("chain failure overwrote output")
	}
}
