package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFloatingUniqueNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/floating-unique.json"))
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
			values := []any{0.0, -0.0}
			if nested {
				props["defaultNested"].(map[string]any)["default"] = []any{values}
			} else {
				props["defaultDoubles"].(map[string]any)["default"] = values
			}
		})
		for _, flags := range [][]string{{}, {"--check"}} {
			fails(t, p.generate(t, append([]string{"--validation"}, flags...)...), 1, "E_SCHEMA_INVALID")
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("floating duplicate default changed old output")
			}
		}
	}
}

func TestFloatingUniqueOriginalCollectionMutationGraduation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/collection-unique.json"))
	put(t, p.Input, bytes.Replace(input, []byte(`"integer"`), []byte(`"number"`), 1))
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("floating graduation check changed output")
	}
}
