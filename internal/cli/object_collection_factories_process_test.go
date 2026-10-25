package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestObjectCollectionFactoriesNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/object-collection-factories.json"))
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
			for _, f := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
					t.Fatal("native/core factory parity", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json"}}
			if checked {
				commands[2] = append(commands[2], "--validation")
			}
			for _, cmd := range commands {
				result(t, invoke(t, p.Root, cmd...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("factory readonly/noop changed output")
			}
			for _, bad := range []string{"missing", "own", "alias"} {
				var d map[string]any
				if err := json.Unmarshal(valid, &d); err != nil {
					t.Fatal(err)
				}
				components := d["components"].(map[string]any)["schemas"].(map[string]any)
				pointer := "/components/schemas/Rows/default/0"
				switch bad {
				case "missing":
					components["Rows"].(map[string]any)["default"] = []any{map[string]any{}}
				case "own":
					components["Rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["mode"].(map[string]any)["default"] = "future"
					pointer = "/components/schemas/Rows/items/properties/mode/default"
				case "alias":
					components["WrappedRows"].(map[string]any)["default"] = []any{map[string]any{"value": 23}}
					pointer = "/components/schemas/WrappedRows/default/0/value"
				}
				data, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, data)
				for _, cmd := range commands {
					failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("bad factory source/value lost Pointer or output", bad, failure)
					}
				}
			}
		}
	}
}
