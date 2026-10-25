package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestObjectNativeEnumOwnDefaultsNativeParityAndFailurePreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/reference-wrapper-component-defaults.json"))
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			put(t, p.Input, valid)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
					t.Fatal("native/core parity", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json"}}
			if checked {
				commands[2] = append(commands[2], "--validation")
			}
			for _, command := range commands {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("readonly/noop changed output")
			}
			for _, sourceBad := range []bool{false, true} {
				var doc map[string]any
				decoder := json.NewDecoder(bytes.NewReader(valid))
				decoder.UseNumber()
				if err := decoder.Decode(&doc); err != nil {
					t.Fatal(err)
				}
				schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
				pointer := "/components/schemas/NativeNarrow/default/f32"
				if sourceBad {
					schemas["NativeTarget"].(map[string]any)["properties"].(map[string]any)["flag"].(map[string]any)["default"] = "bad"
					pointer = "/components/schemas/NativeTarget/properties/flag/default"
				} else {
					schemas["NativeNarrow"].(map[string]any)["default"] = map[string]any{"f32": json.Number("16777216.5")}
				}
				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, data)
				for _, command := range commands {
					failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("source/value failure changed owned output or Pointer", pointer, failure)
					}
				}
			}
		}
	}
}
