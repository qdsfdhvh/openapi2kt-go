package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaInternalDTOChainNative(t *testing.T) {
	p := newProject(t)
	original := read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-dto-chains.json"))
	put(t, p.Input, original)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("chain no-op changed output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Tree/key~"].(map[string]any)["properties"].(map[string]any)["nodeHop"].(map[string]any)["$ref"] = "#/components/schemas/Tree~1key~0/properties/missing"
	})
	for _, args := range [][]string{{}, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_REF_MISSING")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Tree~1key~0/properties/nodeHop/$ref" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("original hop/failure preservation", ds)
		}
	}
}
