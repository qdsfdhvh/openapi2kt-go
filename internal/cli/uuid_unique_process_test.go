package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestUUIDUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/uuid-unique.json"))
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
	for _, nested := range []bool{false, true} {
		put(t, p.Input, input)
		p.change(t, func(d map[string]any) {
			holder := schemas(d)["Holder"].(map[string]any)
			props := holder["properties"].(map[string]any)
			values := []any{"550e8400-e29b-41d4-a716-446655440000", "550E8400E29B41D4A716446655440000"}
			if nested {
				props["defaultNested"].(map[string]any)["default"] = []any{values}
			} else {
				props["defaultValues"].(map[string]any)["default"] = values
			}
		})
		for _, flags := range [][]string{{}, {"--check"}} {
			fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("equivalent-UUID duplicate default changed old output")
			}
		}
	}
}
