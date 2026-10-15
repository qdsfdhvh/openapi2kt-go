package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnionInlineValuesNativeOutputPreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/union-inline-values.json")))
	args := []string{"--model-name-suffix=Dto"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	warningDenied(t, p.generate(t, append(append([]string{}, args...), "--deny-warnings")...))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning/check changed old output")
	}
	p.change(t, func(d map[string]any) {
		choice := schemas(d)["Direct"].(map[string]any)["oneOf"].([]any)[1].(map[string]any)
		choice["properties"].(map[string]any)["value"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
	})
	fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid second inline variant changed old output")
	}
}
