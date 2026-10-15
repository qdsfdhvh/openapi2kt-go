package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaCollectionOwnedDTONative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-collection-owned-dtos.json")))
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed canonical outputs")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["RootMap"].(map[string]any)["additionalProperties"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported-format"
	})
	for _, args := range [][]string{{}, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/RootMap/additionalProperties/properties/id/format" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("source diagnostic/failure preservation", ds)
		}
	}
}
