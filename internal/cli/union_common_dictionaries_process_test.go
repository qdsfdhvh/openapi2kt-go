package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnionCommonDictionariesNativeOutputPreservation(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		p := newProject(t)
		put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/union-common-dictionaries.json")))
		flags := []string{"--model-name-suffix=Dto", "--model-style=" + style}
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		warningDenied(t, p.generate(t, append(append([]string{}, flags...), "--deny-warnings")...))
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("warning/check changed old output")
		}
		p.change(t, func(d map[string]any) {
			schemas(d)["Mapping"].(map[string]any)["additionalProperties"].(map[string]any)["unexpected"] = true
		})
		fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("bad common dictionary source changed old output")
		}
	}
}
