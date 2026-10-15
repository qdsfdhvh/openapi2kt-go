package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAllOfPrimitiveAggregationOriginal(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "allof-primitive-bounds-supported-envelopes.json")))
	result(t, p.generate(t, "--validation"), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
}

func TestAllOfPrimitiveAggregationOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "allof-primitive-bounds-supported-envelopes.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("noop touched bytes/mtime")
	}
	put(t, p.Input, bytes.Replace(input, []byte(`"minLength": 2`), []byte(`"minLength": -1`), 1))
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "BoundsApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" || *r.Diagnostics[0].Pointer != "/components/schemas/Record/allOf/0/properties/value/minLength" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid aggregate changed old output/manifest/unowned")
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("disabled check touched old output")
	}
	result(t, p.generate(t, "--validation", "--exclude-api", "BoundsApi"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("lost unowned")
	}
}
