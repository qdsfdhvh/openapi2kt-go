package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableProductUnionsNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/nullable-product-unions.json")))
	args := []string{"--model-name-suffix=Dto"}
	result(t, p.generate(t, args...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
	warningDenied(t, p.generate(t, append(append([]string{}, args...), "--deny-warnings")...))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning/check changed old output")
	}
	p.change(t, func(d map[string]any) {
		choice := schemas(d)["Product"].(map[string]any)["allOf"].([]any)[0].(map[string]any)["oneOf"].([]any)[0].(map[string]any)
		choice["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
	})
	fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid nullable source changed old output")
	}
}
