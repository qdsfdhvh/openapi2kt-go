package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNativeEnumUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/native-enum-unique.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check or validate changed existing output")
	}
	fails(t, p.generate(t), 1, "E_REF_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("missing validation opt-in changed output")
	}
	for _, nested := range []bool{false, true} {
		put(t, p.Input, input)
		p.change(t, func(d map[string]any) {
			holder := schemas(d)["Holder"].(map[string]any)
			props := holder["properties"].(map[string]any)
			values := []any{1, 1}
			if nested {
				props["defaultNested"].(map[string]any)["default"] = []any{values}
			} else {
				props["defaultCodes"].(map[string]any)["default"] = values
			}
		})
		for _, flags := range [][]string{{}, {"--check"}} {
			fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("native enum duplicate default changed old output")
			}
		}
	}
}

func TestNativeEnumUniqueOriginalNestedNativeGraduation(t *testing.T) {
	p := newProject(t)
	// Preserve the exact prior native CLI rejection as opt-in acceptance.
	put(t, p.Input, nestedNativeEnumDocument(`{"type":"array","items":{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1]}},"default":[]}`))
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("graduated native check changed output")
	}
}
