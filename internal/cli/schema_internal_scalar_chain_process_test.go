package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaInternalScalarChainNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-scalar-chains.json")))
	r := p.generate(t, "--validation", "--model-name-suffix", "Dto")
	t.Logf("actual native CLI code=%d stdout=%s stderr=%s", r.Code, r.Stdout, r.Stderr)
	result(t, r, 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("chain no-op changed output")
	}
	p.change(t, func(doc map[string]any) {
		props := schemas(doc)["Source"].(map[string]any)["properties"].(map[string]any)
		props["leaf/key~"].(map[string]any)["minLength"] = -1
	})
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Source/properties/leaf~1key~0/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("chain source/failure preservation", ds)
		}
	}
}
