package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaDTOAliasChainNative(t *testing.T) {
	for _, kind := range []string{"named", "internal"} {
		t.Run(kind, func(t *testing.T) {
			p := newProject(t)
			put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-"+kind+"-dto-alias-chains.json")))
			result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
			before := snapshot(t, p.Output)
			result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
			result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("alias chain no-op mutated output")
			}
			p.change(t, func(doc map[string]any) {
				schemas(doc)["NodeAlias"].(map[string]any)["$ref"] = "#/components/schemas/Missing"
			})
			for _, args := range [][]string{{}, {"--check"}} {
				ds := fails(t, p.generate(t, args...), 1, "E_REF_MISSING")
				if *ds.Diagnostics[0].Pointer != "/components/schemas/NodeAlias/$ref" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("alias source/failure preservation", ds)
				}
			}
		})
	}
}
