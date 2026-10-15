package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootRawAliasNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "root-raw-aliases.json")))
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
		schemas(doc)["ZOther"].(map[string]any)["allOf"].([]any)[0].(map[string]any)["required"] = false
	})
	for _, args := range [][]string{{"--model-name-suffix", "Dto"}, {"--model-name-suffix", "Dto", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/ZOther/allOf/0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("original target source/output preserved", r)
		}
	}
}
