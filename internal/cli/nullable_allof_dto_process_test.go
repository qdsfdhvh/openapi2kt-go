package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableAllOfDTONativeOriginalSourcePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/nullable-allof-dtos.json")))
	for _, style := range []string{"data_class", "plain_class"} {
		args := []string{"--model-name-suffix=Dto", "--model-style=" + style}
		result(t, p.generate(t, args...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, args...), "--check")...), 0)
		warningDenied(t, p.generate(t, append(append([]string{}, args...), "--deny-warnings")...))
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("check/warning changed old outputs")
		}
	}
	before := snapshot(t, p.Output)
	p.change(t, func(d map[string]any) {
		schemas(d)["Base"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["minLength"] = -1
	})
	failure := fails(t, p.generate(t, "--model-name-suffix=Dto", "--validation"), 1, "E_SCHEMA_INVALID")
	if *failure.Diagnostics[0].Pointer != "/components/schemas/Base/properties/id/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal(failure)
	}
}
