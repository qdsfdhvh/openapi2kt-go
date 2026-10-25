package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestNullConstObjectDefaultsNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/null-const-object-defaults.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			put(t, p.Input, input)
			flags := []string{"--model-name-suffix", "Dto", "--model-style", string(style)}
			if checked {
				flags = append(flags, "--validation")
			}
			result(t, p.generate(t, flags...), 0)
			expected, e := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if e != nil {
				t.Fatal(e)
			}
			for _, f := range expected.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
					t.Fatal("native/core nullconst default parity", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
			result(t, p.generate(t, flags...), 0)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
			if checked {
				validate = append(validate, "--validation")
			}
			result(t, invoke(t, p.Root, validate...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("readonly/noop changed output")
			}
			for _, bad := range []string{"field", "factory"} {
				var doc map[string]any
				if e := json.Unmarshal(input, &doc); e != nil {
					t.Fatal(e)
				}
				schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
				pointer := "/components/schemas/Holder/properties/named/default/fixed"
				if bad == "field" {
					schemas["Holder"].(map[string]any)["properties"].(map[string]any)["named"].(map[string]any)["default"].(map[string]any)["fixed"] = "null"
				} else {
					schemas["Seed"].(map[string]any)["default"].(map[string]any)["fixed"] = "null"
					pointer = "/components/schemas/Seed/default/fixed"
				}
				value, e := json.Marshal(doc)
				if e != nil {
					t.Fatal(e)
				}
				put(t, p.Input, value)
				failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("nullconst default failure lost Pointer/old output", bad, failure)
				}
			}
		}
	}
}
