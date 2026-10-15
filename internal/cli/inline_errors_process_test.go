package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestInlineErrorsNativePreservesOutputOnInvalidOriginalDefault(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/inline-errors.json")))
	args := []string{"--model-name-suffix=Dto"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	result(t, p.generate(t, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("default check/no-op touched files")
	}
	p.change(t, func(d map[string]any) {
		d["paths"].(map[string]any)["/a"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["400"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)["message"].(map[string]any)["nullable"] = "invalid"
	})
	for _, checked := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if checked {
			invocation = append(invocation, "--check")
		}
		report := fails(t, p.generate(t, invocation...), 1, "E_SCHEMA_INVALID")
		if *report.Diagnostics[0].Pointer != "/paths/~1a/get/responses/400/content/application~1json/schema/properties/message/nullable" {
			t.Fatal(report)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid default changed old output")
		}
	}
}
