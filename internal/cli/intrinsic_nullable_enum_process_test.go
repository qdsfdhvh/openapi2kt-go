package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestIntrinsicNullableEnumNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "intrinsic-nullable-enums.json")))
	for _, flag := range []bool{false, true} {
		args := []string{"--model-name-suffix", "Dto"}
		if flag {
			args = append(args, "--validation")
		}
		result(t, p.generate(t, args...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(args, "--check")...), 0)
		result(t, p.generate(t, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("no-op/check")
		}
	}
	put(t, filepath.Join(p.Output, "custom.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) {
		schemas(doc)["ZStatus"].(map[string]any)["enum"] = []any{"ready", "ready"}
	})
	for _, args := range [][]string{{"--validation", "--model-name-suffix", "Dto"}, {"--validation", "--model-name-suffix", "Dto", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/ZStatus/enum/1" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("source/output preservation", r)
		}
	}
}
