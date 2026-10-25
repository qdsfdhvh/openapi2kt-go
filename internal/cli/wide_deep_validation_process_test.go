package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestWideDeepValidationNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/wide-deep-validation.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, valid)
		args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, style)
		result(t, invoke(t, p.Root, args...), 0)
		want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range want.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("allOf numeric native/core parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--validation", "--diagnostic-format=json"}}
		for _, cmd := range commands {
			result(t, invoke(t, p.Root, cmd...), 0)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed output")
		}
		for _, bad := range []string{"outer", "other-bound", "source"} {
			var d map[string]any
			decoder := json.NewDecoder(bytes.NewReader(valid))
			decoder.UseNumber()
			if err := decoder.Decode(&d); err != nil {
				t.Fatal(err)
			}
			components := d["components"].(map[string]any)["schemas"].(map[string]any)
			model := components["WideSixteen"].(map[string]any)
			pointer := "/components/schemas/WideSixteen/default/p199" + strings.Repeat("/0", 16)
			var literal any = 0
			if bad == "other-bound" {
				literal = 1
			}
			for range 16 {
				literal = []any{literal}
			}
			if bad == "source" {
				node := model["properties"].(map[string]any)["p199"].(map[string]any)
				for range 16 {
					node = node["items"].(map[string]any)
				}
				node["minimum"] = "bad"
				pointer = "/components/schemas/WideSixteen/properties/p199" + strings.Repeat("/items", 16) + "/minimum"
			} else {
				wire := "p199"
				if bad == "other-bound" {
					wire = "p198"
					pointer = "/components/schemas/WideSixteen/default/p198" + strings.Repeat("/0", 16)
				}
				model["default"].(map[string]any)[wire] = literal
			}
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, cmd := range commands {
				failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Numeric default failure lost original Pointer or output", bad, failure)
				}
			}
		}
	}
}
