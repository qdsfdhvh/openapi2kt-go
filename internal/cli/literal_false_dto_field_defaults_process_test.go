package cli_test

import (
	"encoding/json"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLiteralFalseDTOFieldDefaultsNativeParity(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/literal-false-dto-field-defaults.json"))
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			put(t, p.Input, input)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
					t.Fatal("native/core factory parity", f.Path)
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
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings", "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("factory readonly/noop changed output")
			}
			var doc map[string]any
			if err := json.Unmarshal(input, &doc); err != nil {
				t.Fatal(err)
			}
			doc["components"].(map[string]any)["schemas"].(map[string]any)["Optional"].(map[string]any)["properties"].(map[string]any)["value"].(map[string]any)["default"] = map[string]any{"blocked": nil}
			bad, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bad)
			for _, command := range commands {
				failed := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/components/schemas/Optional/properties/value/default/blocked" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("factory value refusal lost source/output", failed)
				}
			}
		}
	}
}
