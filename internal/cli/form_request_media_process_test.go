package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestFormRequestMediaNativeParity(t *testing.T) {
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
				if string(read(t, filepath.Join(p.Output, filepath.FromSlash(f.Path)))) != f.Content {
					t.Fatal("core/native form parity", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
			if checked {
				validate = append(validate, "--validation")
			}
			commands := [][]string{args, append(append([]string{}, args...), "--check"), validate}
			for _, c := range commands {
				result(t, invoke(t, p.Root, c...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("form readonly/noop changed output")
			}
			for _, mutation := range []string{"nullable", "optional", "format", "encoding"} {
				var d map[string]any
				if err := json.Unmarshal(valid, &d); err != nil {
					t.Fatal(err)
				}
				body := d["paths"].(map[string]any)["/submit"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)
				media := body["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)
				schema := media["schema"].(map[string]any)
				pointer := "/paths/~1submit/post/requestBody/content/application~1x-www-form-urlencoded"
				code := "E_OPERATIONS_UNSUPPORTED"
				switch mutation {
				case "nullable":
					schema["nullable"] = true
					pointer += "/schema/nullable"
				case "optional":
					body["required"] = false
					pointer = "/paths/~1submit/post/requestBody/required"
				case "format":
					schema["properties"].(map[string]any)["text"].(map[string]any)["format"] = "unsupported"
					pointer += "/schema/properties/text/format"
					code = "E_SCHEMA_UNSUPPORTED"
				case "encoding":
					media["encoding"] = map[string]any{}
					pointer += "/encoding"
				}
				bad, err := json.Marshal(d)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, bad)
				if mutation == "optional" {
					// Preserve this exact formerly rejected mutation as a positive;
					// use a separate root so the original snapshot stays reviewable.
					positive := newProject(t)
					put(t, positive.Input, bad)
					optionalArgs := objectDefaultReferencedPropertyArgs(positive.Input, positive.Output, checked, style)
					result(t, invoke(t, positive.Root, optionalArgs...), 0)
					result(t, invoke(t, positive.Root, append(optionalArgs, "--check")...), 0)
					result(t, invoke(t, p.Root, validate...), 0)
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("positive migration changed original output root")
					}
					continue
				}
				for _, c := range commands {
					fail := fails(t, invoke(t, p.Root, c...), 1, code)
					if fail.Diagnostics[0].Pointer == nil || *fail.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("form source/failure preservation", mutation, fail.Diagnostics)
					}
				}
			}
		}
	}
}
