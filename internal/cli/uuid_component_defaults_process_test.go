package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestUUIDComponentDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		document := read(t, filepath.Join(repo, "tests/fixtures/http/uuid-component-defaults.json"))
		put(t, p.Input, document)
		args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, style)
		result(t, invoke(t, p.Root, args...), 0)
		report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range report.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
				t.Fatal("native/core UUID factory mismatch", file.Path)
			}
		}
		before := snapshot(t, p.Output)
		for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}} {
			result(t, invoke(t, p.Root, command...), 0)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("UUID no-op/check/validate mutated output")
		}
		for _, tc := range []struct {
			name, key, pointer string
			value              any
		}{
			{"Text", "default", "/components/schemas/Text/default", 1},
			{"Outer", "default", "/components/schemas/Outer/default", nil},
			{"BoundedAlias", "default", "/components/schemas/BoundedAlias/default", "UPPER"},
			{"Bounded", "minLength", "/components/schemas/Bounded/minLength", -1},
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
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}} {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("UUID failure lost Pointer/old output", tc, failure)
				}
			}
		}
	}
}
