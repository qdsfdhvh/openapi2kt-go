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
				t.Fatal("native/core UUID-default mismatch", file.Path)
			}
		}
		before := snapshot(t, p.Output)
		commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}}
		for _, command := range commands {
			result(t, invoke(t, p.Root, command...), 0)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("UUID no-op/check/validate changed output")
		}
		for _, tc := range []struct {
			name, key string
			value     any
		}{
			{"Id", "default", 1},
			{"Id", "default", nil},
			{"Bounded", "default", "x"},
			{"Outer", "default", nil},
			{"Outer", "default", "UPPER"},
			{"Text", "default", false},
			{"NullId", "default", false},
			{"Narrow", "default", nil},
			{"Bounded", "minLength", -1},
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
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/"+tc.name+"/"+tc.key || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid UUID default/source lost Pointer or changed output", tc.name, tc.key, failure.Diagnostics)
				}
			}
		}
		var badSource map[string]any
		if err := json.Unmarshal(document, &badSource); err != nil {
			t.Fatal(err)
		}
		badSource["components"].(map[string]any)["schemas"].(map[string]any)["UuidSource"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["minLength"] = -1
		encoded, err := json.Marshal(badSource)
		if err != nil {
			t.Fatal(err)
		}
		put(t, p.Input, encoded)
		for _, command := range commands {
			failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/UuidSource/properties/id/minLength" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("borrowed UUID source diagnostic/output drift", failure.Diagnostics)
			}
		}
	}
}
