package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestComposedUnionTerminalsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/composed-union-terminals.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		product := schemas(d)["Product"].(map[string]any)
		branch := product["allOf"].([]any)[0].(map[string]any)["oneOf"].([]any)[0].(map[string]any)
		branch["properties"].(map[string]any)["body"].(map[string]any)["format"] = "invalid"
	})
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid product source modified output")
	}
}
