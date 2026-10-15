package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableObjectReferenceMembersNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/nullable-object-reference-members.json")))
	args := []string{"--model-name-suffix=Dto"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	warningDenied(t, p.generate(t, append(append([]string{}, args...), "--deny-warnings")...))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning/check changed old output")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["OriginalProduct"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
	})
	fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid referenced nullable source changed old output")
	}
}
