package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMapUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/map-unique.json"))
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
			values := []any{map[string]any{"a": "one", "b": nil}, map[string]any{"b": nil, "a": "one"}}
			if nested {
				props["defaultNested"].(map[string]any)["default"] = []any{values}
			} else {
				props["defaultValues"].(map[string]any)["default"] = values
			}
		})
		for _, flags := range [][]string{{}, {"--check"}} {
			fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("equivalent-Map duplicate default changed old output")
			}
		}
	}
	for _, tc := range []struct {
		field  string
		values []any
	}{
		{"defaultInstantMaps", []any{map[string]any{"a": "2026-01-02T03:04:05Z"}, map[string]any{"a": "2026-01-02T05:04:05+02:00"}}},
		{"defaultInstantMaps", []any{map[string]any{"a": "invalid"}}},
	} {
		put(t, p.Input, input)
		p.change(t, func(d map[string]any) {
			schemas(d)["Holder"].(map[string]any)["properties"].(map[string]any)[tc.field].(map[string]any)["default"] = tc.values
		})
		for _, flags := range [][]string{{}, {"--check"}} {
			fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("special Map invalid default changed old output")
			}
		}
	}
}
