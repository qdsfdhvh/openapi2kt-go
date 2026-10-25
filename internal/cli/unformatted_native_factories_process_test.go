package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestUnformattedNativeFactoriesNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/unformatted-native-factories.json"))
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
					t.Fatal("native/core enum factory parity", f.Path)
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
				t.Fatal("readonly/noop changed output")
			}
			for _, bad := range []string{"Code", "Rounded", "WrappedCode"} {
				var d map[string]any
				decoder := json.NewDecoder(bytes.NewReader(valid))
				decoder.UseNumber()
				if err := decoder.Decode(&d); err != nil {
					t.Fatal(err)
				}
				components := d["components"].(map[string]any)["schemas"].(map[string]any)
				value := json.Number("0")
				if bad == "Rounded" {
					value = json.Number("9007199254740992")
				}
				components[bad].(map[string]any)["default"] = value
				data, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, data)
				for _, cmd := range commands {
					failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/"+bad+"/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("native enum failure lost original Pointer or output", bad, failure)
					}
				}
			}
		}
	}
}
