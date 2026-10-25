package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestOptionalFormRequestNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/form-request-media.json"))
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
			for _, bad := range []string{"root-nullable", "field-default", "field-nullable"} {
				var doc map[string]any
				if err := json.Unmarshal(valid, &doc); err != nil {
					t.Fatal(err)
				}
				schema := doc["paths"].(map[string]any)["/optional-submit"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)
				pointer := "/paths/~1optional-submit/post/requestBody/content/application~1x-www-form-urlencoded/schema"
				switch bad {
				case "root-nullable":
					schema["nullable"] = true
					pointer += "/nullable"
				case "field-default":
					schema["properties"].(map[string]any)["text"].(map[string]any)["default"] = "text"
					pointer += "/properties/text/default"
				case "field-nullable":
					schema["properties"].(map[string]any)["text"].(map[string]any)["nullable"] = true
					pointer += "/properties/text/nullable"
				}
				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, data)
				for _, cmd := range commands {
					failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_OPERATIONS_UNSUPPORTED")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("form source failure changed output/Pointer", bad, failure)
					}
				}
			}
		}
	}
}
