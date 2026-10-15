package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAliasMethodErasureNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "alias-method-erasure.json")))
	result(t, p.generate(t, "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	for _, args := range [][]string{{"--model-name-suffix", "Dto", "--check"}, {"--model-name-suffix", "Dto"}} {
		result(t, p.generate(t, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("alias method no-op changed bytes or mtimes")
		}
	}
	api := string(read(t, filepath.Join(p.Output, "org", "example", "catalog", "api", "DefaultApi.kt")))
	if !strings.Contains(api, "public suspend fun send5(") || !strings.Contains(api, "public suspend fun map2(") {
		t.Fatal("native CLI must allocate underlying List/Map signatures")
	}
}

func TestArrayComponentNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "array-components.json")))
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("array component no-op changed output")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["ZValues"].(map[string]any)["items"].(map[string]any)["minLength"] = -1
	})
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/ZValues/items/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("array source/failure preservation", ds)
		}
	}
}
