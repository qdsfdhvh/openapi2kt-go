package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnionNominalOwnersNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/union-nominal-owners.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["UnionBox"].(map[string]any)["properties"].(map[string]any)["choice"].(map[string]any)["oneOf"].([]any)[0].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "invalid"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid union target modified output")
	}
}
