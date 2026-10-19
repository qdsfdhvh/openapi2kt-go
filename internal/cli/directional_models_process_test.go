package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDirectionalModelsNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "directional-models.json"))
	for _, style := range []string{"data_class", "plain_class"} {
		p := newProject(t)
		put(t, p.Input, input)
		flags := []string{"--model-name-suffix", "Dto", "--model-style", style, "--validation"}
		result(t, p.generate(t, flags...), 0)
		expected, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Format: core.JSON, SourceName: "catalog.json", Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: core.ModelStyle(style), Validation: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, artifact := range expected.Artifacts.Files {
			if actual := string(read(t, filepath.Join(p.Output, filepath.FromSlash(artifact.Path)))); actual != artifact.Content {
				t.Fatal("core/native directional mismatch", artifact.Path)
			}
		}
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		result(t, p.generate(t, flags...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("directional check/no-op changed output")
		}
		// Even a response-only hidden field must validate before committing.
		bad := strings.Replace(string(input), `"writeOnly": true`, `"writeOnly": "invalid"`, 1)
		put(t, p.Input, []byte(bad))
		fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("directional rejection damaged old output")
		}
	}
}
