package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBigintReferenceChainsNativeOriginalSourceAndPreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/bigint-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) { schemas(d)["ZAmount"].(map[string]any)["minimum"] = 0 })
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid bigint source replaced accepted output")
	}
}
