package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableIntersectionNativeOutputAndSuppression(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-product-unions.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("intersection check/validate mutated output")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["JointA"].(map[string]any)["properties"].(map[string]any)["left"].(map[string]any)["format"] = "unsupported"
	})
	for _, flags := range [][]string{{}, {"--check"}, {"--exclude-api", "DefaultApi"}} {
		fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid intersection member changed output")
		}
	}
}
