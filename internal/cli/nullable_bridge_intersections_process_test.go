package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestNullableBridgeIntersectionsNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-bridge-intersections.json"))
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
					t.Fatal("native/core form parity", f.Path)
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
				t.Fatal("readonly/noop changed form output")
			}
			for _, bad := range []string{"composed", "bad-format", "missing"} {
				var doc map[string]any
				if e := json.Unmarshal(valid, &doc); e != nil {
					t.Fatal(e)
				}
				catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
				code := "E_REF_UNSUPPORTED"
				pointer := "/components/schemas/Consumer/allOf/0"
				switch bad {
				case "composed":
					value := catalog["Value"].(map[string]any)
					value["allOf"] = []any{map[string]any{"properties": value["properties"]}}
					delete(value, "properties")
				case "bad-format":
					catalog["Value"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
					code = "E_SCHEMA_UNSUPPORTED"
					pointer = "/components/schemas/Value/properties/id/format"
				case "missing":
					catalog["Hops"].(map[string]any)["properties"].(map[string]any)["first"].(map[string]any)["$ref"] = "#/components/schemas/Missing"
					code = "E_REF_MISSING"
					pointer = "/components/schemas/Hops/properties/first/$ref"
				}
				data, e := json.Marshal(doc)
				if e != nil {
					t.Fatal(e)
				}
				put(t, p.Input, data)
				for _, cmd := range commands {
					failure := fails(t, invoke(t, p.Root, cmd...), 1, code)
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("bridge failure lost Pointer/old output", bad, failure)
					}
				}
			}

		}
	}
}
