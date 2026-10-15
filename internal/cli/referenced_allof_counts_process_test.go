package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReferencedNestedAllOfCountsNativeCheckAndFailurePreservation(t *testing.T) {
	p := newProject(t)
	original := read(t, filepath.Join(repo, "tests/fixtures/http/referenced-allof-counts.json"))
	put(t, p.Input, original)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Count check changed owned output")
	}
	at := bytes.Index(original, []byte(`"RecordUpper": {`))
	if at < 0 {
		t.Fatal("Missing original upper source")
	}
	changed := append(bytes.Clone(original[:at]), bytes.Replace(original[at:], []byte(`"maxProperties": 3`), []byte(`"maxProperties": 1`), 1)...)
	put(t, p.Input, changed)
	failure := fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/RecordUpper/maxProperties" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Original whole-object source failure changed output", failure)
	}
	put(t, p.Input, original)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Default-off count failure changed output")
	}
}
