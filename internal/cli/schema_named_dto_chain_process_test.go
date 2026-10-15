package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaNamedDTOChainNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-named-dto-chains.json")))
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op mutated canonical output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Tree/key~"].(map[string]any)["properties"].(map[string]any)["node"].(map[string]any)["$ref"] = "#/components/schemas/Missing"
	})
	for _, args := range [][]string{{}, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_REF_MISSING")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Tree~1key~0/properties/node/$ref" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("failure source/old output", ds)
		}
	}
}
