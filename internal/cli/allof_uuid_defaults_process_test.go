package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestAllOfUUIDDefaultsNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/allof-uuid-defaults.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, valid)
		args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, style)
		result(t, invoke(t, p.Root, args...), 0)
		want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range want.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("allOf UUID native/core parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--validation", "--diagnostic-format=json"}}
		for _, cmd := range commands {
			result(t, invoke(t, p.Root, cmd...), 0)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed output")
		}
		for _, bad := range []string{"outer", "original-own", "alias"} {
			var d map[string]any
			if err := json.Unmarshal(valid, &d); err != nil {
				t.Fatal(err)
			}
			components := d["components"].(map[string]any)["schemas"].(map[string]any)
			pointer := "/components/schemas/Factory/default/value"
			switch bad {
			case "outer":
				components["Factory"].(map[string]any)["default"] = map[string]any{"value": "00000000-0000-0000-0000-000000000000"}
			case "original-own":
				components["Lower"].(map[string]any)["properties"].(map[string]any)["value"].(map[string]any)["default"] = "00000000-0000-0000-0000-000000000000"
				pointer = "/components/schemas/Lower/properties/value/default"
			case "alias":
				components["NarrowFactory"].(map[string]any)["default"] = map[string]any{"value": "00000000-0000-0000-0000-000000000000"}
				pointer = "/components/schemas/NarrowFactory/default/value"
			}
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, cmd := range commands {
				failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("UUID default failure lost original Pointer or output", bad, failure)
				}
			}
		}
	}
}
