package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaInternalScalarPointerQualifiedNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-scalar-pointers-qualified.json")))
	actual := p.generate(t, "--validation")
	t.Logf("actual native CLI exit=%d stdout=%s stderr=%s", actual.Code, actual.Stdout, actual.Stderr)
	result(t, actual, 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed output")
	}
}

func TestSchemaInternalScalarPointerNativeFailurePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-scalar-pointers-qualified.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Source"].(map[string]any)["properties"].(map[string]any)["a/b~c"].(map[string]any)["minLength"] = -1
	})
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "PointersApi"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/Source/properties/a~1b~0c/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("lost original source or modified prior output", r)
		}
	}
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "schema-internal-scalar-pointers.json")))
	r := fails(t, p.generate(t, "--validation"), 1, "E_NAME_INVALID")
	if *r.Diagnostics[0].Pointer != "/components/schemas/Source/properties/" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("frozen unsupported host changed", r)
	}
	put(t, p.Input, input)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
}
