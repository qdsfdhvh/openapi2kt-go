package cli_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestFalseNominalSignatureNativePreservesOutput(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		p := newProject(t)
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = false
		}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide/~": map[string]any{"properties": props}}}}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		put(t, p.Input, data)
		flags := []string{"--model-style", style}
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		pkg := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
		args := []string{"generate", "--input", p.Input, "--output", p.Output, "--model-style", style, "--package-name", pkg, "--diagnostic-format=json"}
		for _, cmd := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--model-style", style, "--package-name", pkg, "--diagnostic-format=json"}} {
			failed := fails(t, invoke(t, p.Root, cmd...), 1, "E_MODEL_LIMIT")
			if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/components/schemas/Wide~1~0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("descriptor refusal lost original source/output", failed)
			}
		}
	}
}
