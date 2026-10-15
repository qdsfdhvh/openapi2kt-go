package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaInternalDTONative(t *testing.T) {
	p := newProject(t)
	original := read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-dtos.json"))
	put(t, p.Input, original)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed owned output")
	}
	p.change(t, func(doc map[string]any) {
		props := schemas(doc)["Tree/key~"].(map[string]any)["properties"].(map[string]any)
		props["node"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unknown"
	})
	for _, args := range [][]string{{}, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Tree~1key~0/properties/node/properties/id/format" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid target changed old output")
		}
	}
}
