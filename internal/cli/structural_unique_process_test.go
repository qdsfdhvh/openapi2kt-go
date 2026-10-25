package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestStructuralUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/map-unique.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("structural no-op changed owned or handwritten files")
	}
	p.change(t, func(d map[string]any) {
		field := schemas(d)["Holder"].(map[string]any)["properties"].(map[string]any)["defaultListRows"].(map[string]any)
		field["default"] = []any{[]any{1}, []any{1}}
	})
	for _, flags := range [][]string{{}, {"--check"}} {
		fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("duplicate structural default changed old output")
		}
	}
}
