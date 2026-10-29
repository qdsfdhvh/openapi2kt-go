package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

// Exact previously rejected defaults remain positive native migrations, while
// the original no-default output roots stay unchanged.
func formFieldDefaultMigration(t *testing.T, input []byte, checked bool, style core.ModelStyle) {
	t.Helper()
	p := newProject(t)
	put(t, p.Input, input)
	args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	result(t, invoke(t, p.Root, append(args, "--check")...), 0)
	result(t, invoke(t, p.Root, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("default migration changed output on check/no-op")
	}
}

func TestFormFieldDefaultsNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/form-field-defaults.json"))
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			put(t, p.Input, input)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, filepath.FromSlash(f.Path)))) != f.Content {
					t.Fatal("default form core/native divergence", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
			if checked {
				validate = append(validate, "--validation")
			}
			commands := [][]string{args, append(append([]string{}, args...), "--check"), validate}
			for _, cmd := range commands {
				result(t, invoke(t, p.Root, cmd...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("readonly/default no-op changed output")
			}
			var doc map[string]any
			decoder := json.NewDecoder(bytes.NewReader(input))
			decoder.UseNumber()
			if err := decoder.Decode(&doc); err != nil {
				t.Fatal(err)
			}
			schema := doc["paths"].(map[string]any)["/submit"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)
			schema["properties"].(map[string]any)["code"].(map[string]any)["default"] = "unknown"
			bad, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bad)
			for _, cmd := range commands {
				failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/paths/~1submit/post/requestBody/content/application~1x-www-form-urlencoded/schema/properties/code/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("bad default lost source/old output", failure)
				}
			}
		}
	}
}
