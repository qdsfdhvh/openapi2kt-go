package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRawResponseMediaNativeParity(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "raw-response-media.json"))
	p := newProject(t)
	put(t, p.Input, input)
	result(t, p.generate(t), 0)
	want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Format: core.JSON, SourceName: "catalog.json", Options: core.GenerationOptions{PackageName: "org.example.catalog"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range want.Artifacts.Files {
		if string(read(t, filepath.Join(p.Output, filepath.FromSlash(artifact.Path)))) != artifact.Content {
			t.Fatal("native raw-media output differs", artifact.Path)
		}
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("raw-media check modified files")
	}
	bad := strings.Replace(string(input), `"format": "binary"`, `"format": "unsupported"`, 1)
	put(t, p.Input, []byte(bad))
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("raw-media rejection damaged existing output")
	}
}
