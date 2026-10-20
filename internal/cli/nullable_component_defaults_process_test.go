package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Native generation, readonly check, core parity and failure preservation match
// the plan for a nullable component root carrying a non-null default.
func TestNullableComponentDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-component-defaults.json"))
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
				// The nullable alias keeps its public type and gains the factory.
				alias := read(t, p.Output+"/org/example/catalog/models/NameDto.kt")
				if !strings.Contains(string(alias), "public typealias NameDto = kotlin.String?") ||
					!strings.Contains(string(alias), "public fun `NameDto default`()") {
					t.Fatal("nullable factory missing from native output", string(alias))
				}

				nullEnum := string(read(t, p.Output+"/org/example/catalog/models/NilModeDto.kt"))
				if !strings.Contains(nullEnum, "public fun `NilModeDto default`(): NilModeDto? = null") {
					t.Fatal("nullable enum null factory missing", nullEnum)
				}

				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("readonly check changed bytes or mtimes")
				}

				// A bad annotation reports inside the component and leaves the
				// previously generated output untouched.
				bad := bytes.Replace(document, []byte(`"default": "fallback"`), []byte(`"default": 5`), 1)
				if bytes.Equal(bad, document) {
					t.Fatal("missing original annotation")
				}
				put(t, p.Input, bad)
				failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				want := "/components/schemas/Name/default"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != want {
					t.Fatalf("bad nullable default reported %v, want %s", failure.Diagnostics[0].Pointer, want)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("bad nullable default changed owned outputs")
				}
				badNull := bytes.Replace(document, []byte(`"type": ["string", "null"], "default": null`), []byte(`"type": "string", "default": null`), 1)
				if bytes.Equal(badNull, document) {
					t.Fatal("null-factory fixture annotation missing")
				}
				put(t, p.Input, badNull)
				for _, command := range [][]string{args, append(append([]string{}, args...), "--check")} {
					failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/NilName/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("nonnullable null factory did not preserve output", failure)
					}
				}
			})
		}
	}
}
