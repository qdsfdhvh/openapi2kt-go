package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestFalseDTOFieldDefaultsNativeParity(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/false-dto-field-defaults.json"))
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
				if string(read(t, p.Output+"/"+f.Path)) != f.Content {
					t.Fatal("native/core parity", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
			if checked {
				validate = append(validate, "--validation")
			}
			commands := [][]string{args, append(append([]string{}, args...), "--check"), validate}
			for _, command := range commands {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("readonly/noop changed output")
			}
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings", "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("false root warning policy changed output")
			}
			var bad map[string]any
			if err := json.Unmarshal(valid, &bad); err != nil {
				t.Fatal(err)
			}
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Alias"].(map[string]any)["nullable"] = "invalid"
			data, err := json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				pointer := "/components/schemas/Alias/nullable"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost original source or existing output", failure.Diagnostics)
				}

			}
			if err := json.Unmarshal(valid, &bad); err != nil {
				t.Fatal(err)
			}
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Consumer"].(map[string]any)["properties"].(map[string]any)["target"].(map[string]any)["default"].(map[string]any)["value"] = 0
			data, err = json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				pointer := "/components/schemas/Consumer/properties/target/default/value"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost default source or existing output", failure.Diagnostics)
				}
			}
		}
	}
}
