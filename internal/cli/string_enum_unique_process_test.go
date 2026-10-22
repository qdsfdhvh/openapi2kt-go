package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestStringEnumUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/string-enum-unique.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check or validate changed existing output")
	}
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("missing validation opt-in changed output")
	}
	for _, values := range [][]any{{"ready", "ready"}, {nil, nil}, {"future"}} {
		for _, field := range []string{"codes", "nested", "refRows", "refNested"} {
			put(t, p.Input, input)
			p.change(t, func(d map[string]any) {
				props := schemas(d)["Holder"].(map[string]any)["properties"].(map[string]any)
				if field == "nested" {
					props[field].(map[string]any)["default"] = []any{values}
				} else if field == "refNested" {
					props[field].(map[string]any)["default"] = []any{map[string]any{"entry": values}}
				} else {
					props[field].(map[string]any)["default"] = values
				}
			})
			for _, flags := range [][]string{{}, {"--check"}} {
				fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid enum uniqueness default changed old output")
				}
			}
		}
	}
}
