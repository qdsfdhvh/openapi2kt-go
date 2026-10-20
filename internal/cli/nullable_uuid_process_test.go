package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestNullableUUIDNativeParityAndPreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			input := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-uuid.json"))
			if !checked {
				var doc map[string]any
				if err := json.Unmarshal(input, &doc); err != nil {
					t.Fatal(err)
				}
				allSchemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
				branches := allSchemas["Aggregate"].(map[string]any)["allOf"].([]any)
				delete(branches[0].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any), "minLength")
				delete(branches[1].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any), "pattern")
				bounded := allSchemas["Bounded"].(map[string]any)
				for _, key := range []string{"minLength", "maxLength", "pattern"} {
					delete(bounded, key)
				}
				var err error
				input, err = json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
			}
			put(t, p.Input, input)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
					t.Fatal("nullable UUID native/core mismatch", file.Path)
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
				t.Fatal("nullable UUID no-op/check/validate changed output")
			}
			var bad map[string]any
			if err := json.Unmarshal(input, &bad); err != nil {
				t.Fatal(err)
			}
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Maybe"].(map[string]any)["default"] = 1
			encoded, err := json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, encoded)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Maybe/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("nullable UUID failure lost original Pointer/output", failure)
				}
			}
		}
	}
}
