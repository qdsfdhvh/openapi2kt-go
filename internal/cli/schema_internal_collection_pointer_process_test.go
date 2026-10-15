package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaInternalCollectionPointerQualifiedNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-collection-pointers.json")))
	actual := p.generate(t, "--validation", "--model-name-suffix", "Dto")
	t.Logf("actual inner native CLI exit=%d stdout=%s stderr=%s", actual.Code, actual.Stdout, actual.Stderr)
	result(t, actual, 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed output")
	}
}

func TestSchemaInternalCollectionPointerNativeFailurePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-collection-pointers.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Source"].(map[string]any)["properties"].(map[string]any)["list"].(map[string]any)["minItems"] = -1
	})
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "CollectionPointersApi"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/Source/properties/list/minItems" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("source/preserved output lost", r)
		}
	}
	put(t, p.Input, input)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
}
