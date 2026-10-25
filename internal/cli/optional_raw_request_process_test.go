package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestOptionalRawRequestNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/raw-request-media.json"))
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			put(t, p.Input, valid)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range want.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
					t.Fatal("native/core parity", f.Path)
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
				t.Fatal("readonly/noop changed output")
			}
			for _, path := range []string{"/optional-text", "/optional-bytes", "/optional-events"} {
				var doc map[string]any
				if err := json.Unmarshal(valid, &doc); err != nil {
					t.Fatal(err)
				}
				content := doc["paths"].(map[string]any)[path].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)
				media := "text/plain"
				schema := map[string]any{"type": "string", "nullable": true}
				encodedMedia := "text~1plain"
				if path == "/optional-bytes" {
					media = "application/octet-stream"
					encodedMedia = "application~1octet-stream"
					schema["format"] = "binary"
				}
				content[media].(map[string]any)["schema"] = schema
				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				put(t, p.Input, data)
				encodedPath := "~1" + path[1:]
				for _, cmd := range commands {
					failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_OPERATIONS_UNSUPPORTED")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/paths/"+encodedPath+"/post/requestBody/content/"+encodedMedia+"/schema" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("nullable raw schema lost source or damaged output", failure)
					}
				}
			}
		}
	}
}
