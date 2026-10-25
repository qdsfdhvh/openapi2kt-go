package cli_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJVMDataCopyNativeStyleParityAndFailurePreservation(t *testing.T) {
	var leaf any = map[string]any{"type": "integer", "minimum": 1}
	for range 16 {
		leaf = map[string]any{"type": "array", "items": leaf}
	}
	fields := map[string]any{}
	for i := range 203 {
		fields[fmt.Sprintf("p%d", i)] = leaf
	}
	data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": map[string]any{"properties": fields}}}})
	if e != nil {
		t.Fatal(e)
	}
	p := newProject(t)
	put(t, p.Input, data)
	packageName := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", packageName, "--model-name-suffix", "Dto", "--model-style", "plain_class", "--validation", "--diagnostic-format=json"}
	result(t, invoke(t, p.Root, args...), 0)
	want, e := core.GenerateWithDiagnostics(core.GenerationRequest{Document: data, Options: core.GenerationOptions{PackageName: packageName, ModelNameSuffix: "Dto", ModelStyle: core.PlainClass, Validation: true}})
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range want.Artifacts.Files {
		if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
			t.Fatal("native Plain/core bytes differ", f.Path)
		}
	}
	before := snapshot(t, p.Output)
	result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
	for i, arg := range args {
		if arg == "plain_class" {
			args[i] = "data_class"
		}
	}
	for _, cmd := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--package-name", packageName, "--model-style", "data_class", "--validation", "--diagnostic-format=json"}} {
		failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_MODEL_LIMIT")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Wide" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("copy limit changed old output/source Pointer", failure)
		}
	}
}
