package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestWrapperOwnedDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			document := read(t, filepath.Join(repo, "tests/fixtures/http/wrapper-owned-defaults.json"))
			put(t, p.Input, document)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
					t.Fatal("native/core independent-default mismatch", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json"}} {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("no-op/check/validate mutated output")
			}
			for _, tc := range []struct {
				name, key, code, pointer string
				value                    any
			}{
				{"TargetObject", "default", "E_SCHEMA_INVALID", "/components/schemas/TargetObject/default", map[string]any{}},
				{"MiddleObject", "default", "E_SCHEMA_INVALID", "/components/schemas/MiddleObject/default", map[string]any{}},
				{"OuterObject", "default", "E_SCHEMA_INVALID", "/components/schemas/OuterObject/default", map[string]any{}},
				{"MiddleMap", "default", "E_SCHEMA_INVALID", "/components/schemas/MiddleMap/default/a~1~0", map[string]any{"a/~": false}},
				{"BridgeObject", "default", "E_SCHEMA_INVALID", "/components/schemas/BridgeObject/default", map[string]any{}},
				{"NullObject", "nullable", "E_SCHEMA_INVALID", "/components/schemas/NullObject/default", false},
			} {
				var bad map[string]any
				if err := json.Unmarshal(document, &bad); err != nil {
					t.Fatal(err)
				}
				bad["components"].(map[string]any)["schemas"].(map[string]any)[tc.name].(map[string]any)[tc.key] = tc.value
				encoded, err := json.Marshal(bad)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, encoded)
				for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json"}} {
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
