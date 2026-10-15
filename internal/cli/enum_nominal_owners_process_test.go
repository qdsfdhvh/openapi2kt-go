package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnumNominalOwnersNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/enum-nominal-owners.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Owner"].(map[string]any)["properties"].(map[string]any)["state"].(map[string]any)["enum"] = []any{"new", nil}
	})
	// The original nullable ["new", null] vocabulary now keeps the original
	// Owner.State classifier and its nullable consumers rather than failing.
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	before = snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Owner"].(map[string]any)["properties"].(map[string]any)["state"].(map[string]any)["enum"] = []any{"new", nil, nil}
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("malformed enum target changed output")
	}
}
