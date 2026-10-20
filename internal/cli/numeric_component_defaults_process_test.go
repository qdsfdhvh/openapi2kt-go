package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestNumericComponentDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{true} {
			p := newProject(t)
			document := read(t, filepath.Join(repo, "tests/fixtures/http/numeric-component-defaults.json"))
			put(t, p.Input, document)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
					t.Fatal("native/core numeric-default mismatch", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}} {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("no-op/check/validate mutated output")
			}
			for _, tc := range []struct {
				name, key, code, pointer string
				value                    any
			}{
				{"Int32", "default", "E_SCHEMA_INVALID", "/components/schemas/Int32/default", json.Number("2147483648")},
				{"LongMax", "default", "E_SCHEMA_INVALID", "/components/schemas/LongMax/default", json.Number("9223372036854775808")},
				{"FloatValue", "default", "E_SCHEMA_INVALID", "/components/schemas/FloatValue/default", json.Number("3.5e38")},
				{"CountAlias", "default", "E_SCHEMA_INVALID", "/components/schemas/CountAlias/default", 3},
				{"Outer", "default", "E_SCHEMA_INVALID", "/components/schemas/Outer/default", nil},
				{"Count", "multipleOf", "E_SCHEMA_INVALID", "/components/schemas/Count/multipleOf", 0},
			} {
				var bad map[string]any
				decoder := json.NewDecoder(bytes.NewReader(document))
				decoder.UseNumber()
				if err := decoder.Decode(&bad); err != nil {
					t.Fatal(err)
				}
				bad["components"].(map[string]any)["schemas"].(map[string]any)[tc.name].(map[string]any)[tc.key] = tc.value
				encoded, err := json.Marshal(bad)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, encoded)
				for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}} {
					failure := fails(t, invoke(t, p.Root, command...), 1, tc.code)
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						details, _ := json.Marshal(failure.Diagnostics)
						t.Fatalf("%s: pointer/output drift: %s", tc.name, details)
					}
				}
			}
		}
	}
}
