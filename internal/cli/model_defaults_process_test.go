package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestModelDefaultsNativePreservesOutputOnInvalidOriginalDefault(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/model-defaults.json")))
	args := []string{"--model-name-suffix=Dto"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	result(t, p.generate(t, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("default check/no-op touched files")
	}
	p.change(t, func(d map[string]any) {
		schemas(d)["Record"].(map[string]any)["properties"].(map[string]any)["count"].(map[string]any)["default"] = "invalid synthetic default"
	})
	for _, checked := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if checked {
			invocation = append(invocation, "--check")
		}
		report := fails(t, p.generate(t, invocation...), 1, "E_SCHEMA_INVALID")
		if *report.Diagnostics[0].Pointer != "/components/schemas/Record/properties/count/default" {
			t.Fatal(report)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid default changed old output")
		}
	}
}
