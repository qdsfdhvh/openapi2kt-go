package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableComposedReferenceMembersNativePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-composed-reference-members.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check or validate changed existing output")
	}
	// Source schema validation still precedes branch materialization and commit.
	bad := bytes.Replace(input, []byte(`"type": "string"`), []byte(`"type": "string", "format": "unsupported"`), 1)
	if bytes.Equal(bad, input) {
		t.Fatal("invalid source control was not injected")
	}
	put(t, p.Input, bad)
	for _, flags := range [][]string{{}, {"--check"}} {
		fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("failed member source changed old output")
		}
	}
}
