package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBytesReferenceChainsNativeOriginalSourcePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/bytes-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) { schemas(d)["ZBytes"].(map[string]any)["default"] = 5 })
	fails(t, p.generate(t, "--model-name-suffix=Dto"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid bytes source changed accepted output")
	}
}
