package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecursiveJSONReferenceGraphNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/recursive-json-reference-graph.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix=Dto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check changed old output")
	}
	p.change(t, func(d map[string]any) { schemas(d)["Through"].(map[string]any)["nullable"] = false })
	// The exact former negative now retains the complete JSON cover. Preserve
	// its original no-suffix invocation as well as the source mutation above.
	generated := result(t, p.generate(t), 0)
	if len(generated.Warnings) != 2 {
		t.Fatal("nullable graph edge changed projection warnings", generated.Warnings)
	}
	positive := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	if !reflect.DeepEqual(positive, snapshot(t, p.Output)) {
		t.Fatal("nullable graph check changed output")
	}
	p.change(t, func(d map[string]any) { schemas(d)["Through"].(map[string]any)["nullable"] = "invalid" })
	failure := fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
	if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Through/nullable" || !reflect.DeepEqual(positive, snapshot(t, p.Output)) {
		t.Fatal("malformed graph intent lost original Pointer or changed old output", failure.Diagnostics)
	}
}
