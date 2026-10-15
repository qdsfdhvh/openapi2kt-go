package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestTemporalReferenceChainsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/temporal-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) { schemas(d)["Stamp"].(map[string]any)["format"] = "unsupported" })
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original temporal modified output")
	}
}
