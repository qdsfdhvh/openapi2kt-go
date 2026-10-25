package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestTupleTargetDefaultsNativeParity(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-target-defaults.json"))
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
			denied := warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			if len(denied.Warnings) != 17 || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("tuple warning policy lost source/output", denied)
			}
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings", "--allow-warning=W_TUPLE_APPROXIMATION", "--allow-warning=W_ALL_OF_APPROXIMATION", "--check")...), 0)
			var bad map[string]any
			if err := json.Unmarshal(valid, &bad); err != nil {
				t.Fatal(err)
			}
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Rows"].(map[string]any)["prefixItems"].([]any)[0].(map[string]any)["nullable"] = "invalid"
			data, err := json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				pointer := "/components/schemas/Rows/prefixItems/0/nullable"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost original source or existing output", failure.Diagnostics)
				}
			}
			for _, tc := range []struct {
				name    string
				value   any
				pointer string
			}{
				{"Mode", "future", "/components/schemas/Mode/default"},
				{"PlainAlias", map[string]any{}, "/components/schemas/PlainAlias/default"},
				{"AllAlias", map[string]any{"value": "valid", "count": "bad"}, "/components/schemas/AllAlias/default/count"},
			} {
				var original map[string]any
				if err := json.Unmarshal(valid, &original); err != nil {
					t.Fatal(err)
				}
				original["components"].(map[string]any)["schemas"].(map[string]any)[tc.name].(map[string]any)["default"] = tc.value
				bad, err := json.Marshal(original)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, bad)
				for _, command := range commands {
					failed := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
					if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("independent target factory failure lost source/output", tc, failed.Diagnostics)
					}
				}
			}
		}
	}
}
