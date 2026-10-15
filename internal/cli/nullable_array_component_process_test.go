package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNullableArrayNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "nullable-array-components.json")))
	options := []string{"--validation", "--model-name-suffix", "Dto"}
	result(t, p.generate(t, options...), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, append(options, "--check")...), 0)
	result(t, p.generate(t, options...), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("nullable aliases changed on no-op/check/validate")
	}
	api := string(read(t, filepath.Join(p.Output, "org/example/catalog/api/DefaultApi.kt")))
	if !strings.Contains(api, "kotlin.collections.List<org.example.catalog.models.RowsDtoInner>") || !strings.Contains(api, "kotlin.collections.List<org.example.catalog.models.StatesDtoInner>") {
		t.Fatal("native narrow view must qualify component item peers", api)
	}
	put(t, filepath.Join(p.Output, "custom.txt"), []byte("synthetic hand-written file"))
	before = snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["minLength"] = -1
	})
	for _, extra := range [][]string{options, append(options, "--check")} {
		r := fails(t, p.generate(t, extra...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/Rows/items/properties/id/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("nullable forward alias source/failure preservation", r)
		}
	}
}
