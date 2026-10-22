package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParameterStylesNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/parameter-styles.json")))
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check or validate changed existing output")
	}
	p.change(t, func(d map[string]any) {
		paths := d["paths"].(map[string]any)
		operation := paths["/deepStrings"].(map[string]any)["get"].(map[string]any)
		operation["parameters"].([]any)[0].(map[string]any)["explode"] = false
	})
	for _, flags := range [][]string{{}, {"--check"}} {
		fails(t, p.generate(t, flags...), 1, "E_OPERATIONS_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid deepObject style changed old output")
		}
	}
}
