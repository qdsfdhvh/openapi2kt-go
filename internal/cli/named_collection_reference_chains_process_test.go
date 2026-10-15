package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNamedCollectionReferenceChainsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/named-collection-reference-chains.json")))
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) { schemas(d)["ZValues"].(map[string]any)["minItems"] = -1 })
	fails(t, p.generate(t, "--model-name-suffix=Dto", "--validation"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("badnamedcollectionsourcechangedoldoutput")
	}
}
