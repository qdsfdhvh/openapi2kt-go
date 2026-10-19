package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Native generation, readonly check, core parity and failure preservation match
// the plan for a raw JSON carrier component root default.
func TestRawJSONComponentDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := read(t, filepath.Join(repo, "tests/fixtures/http/raw-json-component-defaults.json"))
				put(t, p.Input, document)
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)

				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
				}
				open := read(t, p.Output+"/org/example/catalog/models/OpenDto.kt")
				if !strings.Contains(string(open), "public typealias OpenDto = kotlinx.serialization.json.JsonObject") ||
					!strings.Contains(string(open), "public fun `OpenDto default`()") {
					t.Fatal("raw carrier factory missing from native output", string(open))
				}

				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("readonly check changed bytes or mtimes")
				}

				// A wrong-kind annotation reports inside the component and leaves
				// the previously generated output untouched.
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Open":{"type":"object","additionalProperties":true,"default":[1]}}}}`))
				failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				want := "/components/schemas/Open/default"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != want {
					t.Fatalf("bad raw carrier default reported %v, want %s", failure.Diagnostics[0].Pointer, want)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("bad raw carrier default changed owned outputs")
				}
			})
		}
	}
}
