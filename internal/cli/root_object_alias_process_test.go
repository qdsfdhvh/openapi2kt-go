package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootObjectAliasNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "root-object-aliases.json")))
	for _, flag := range []bool{false, true} {
		options := []string{"--model-name-suffix", "Dto"}
		if flag {
			options = append(options, "--validation")
		}
		result(t, p.generate(t, options...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(options, "--check")...), 0)
		result(t, p.generate(t, options...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("root object alias no-op/check changed output")
		}
	}
	put(t, filepath.Join(p.Output, "custom.txt"), []byte("synthetic hand-written file"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) {
		schemas(doc)["ZRecord"].(map[string]any)["properties"].(map[string]any)["id"] = map[string]any{"type": "string", "minLength": -1}
	})
	for _, args := range [][]string{{"--validation", "--model-name-suffix", "Dto"}, {"--validation", "--model-name-suffix", "Dto", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/ZRecord/properties/id/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("root alias errors must preserve original source/output", r)
		}
	}
}
