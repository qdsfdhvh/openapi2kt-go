package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootInternalDTOAliasNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "root-internal-dto-aliases.json")))
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	alias := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "NodeAliasDto.kt"))
	if !strings.Contains(string(alias), "typealias NodeAliasDto = org.example.catalog.models.TreeKeyDto.Node") {
		t.Fatal("native root alias did not reuse physical DTO", string(alias))
	}
	result(t, p.generate(t, "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check/no-op changed bytes/mtime/mode")
	}
	p.change(t, func(doc map[string]any) {
		props := schemas(doc)["Tree/key~"].(map[string]any)["properties"].(map[string]any)
		props["node"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unknown"
	})
	for _, args := range [][]string{{"--model-name-suffix", "Dto"}, {"--model-name-suffix", "Dto", "--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Tree~1key~0/properties/node/properties/id/format" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("root alias target failure changed existing output")
		}
	}
}
