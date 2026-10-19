package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReferencedNestedAllOfBoundsNativeCheckAndFailurePreservation(t *testing.T) {
	p := newProject(t)
	original := read(t, filepath.Join(repo, "tests/fixtures/http/referenced-allof-bounds.json"))
	put(t, p.Input, original)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Check changed owned output")
	}
	// A winning initializer cannot bypass a bound from another source.
	defaultsAt := bytes.Index(original, []byte(`"Defaults": {`))
	if defaultsAt < 0 {
		t.Fatal("Missing default-bearing aggregate")
	}
	invalidDefault := append(bytes.Clone(original[:defaultsAt]), bytes.Replace(original[defaultsAt:], []byte(`"default": "abc"`), []byte(`"default": "a"`), 1)...)
	put(t, p.Input, invalidDefault)
	defaultFailure := fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if defaultFailure.Diagnostics[0].Pointer == nil || *defaultFailure.Diagnostics[0].Pointer != "/components/schemas/Defaults/allOf/1/allOf/0/properties/value/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Borrowed-bound default failure changed old output", defaultFailure)
	}
	// Retain exact unrelated Long extrema; generic map decoding would round
	// JSON numbers before re-encoding and fail at an unrelated source.
	at := bytes.Index(original, []byte(`"RecordUpper": {`))
	if at < 0 {
		t.Fatal("Missing original upper contribution")
	}
	changed := append(bytes.Clone(original[:at]), bytes.Replace(original[at:], []byte(`"maxLength": 3`), []byte(`"maxLength": 1`), 1)...)
	put(t, p.Input, changed)
	failure := fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/RecordUpper/properties/value/maxLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Original source failure changed old output", failure)
	}
	put(t, p.Input, original)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Default-off failure changed old output")
	}
}
