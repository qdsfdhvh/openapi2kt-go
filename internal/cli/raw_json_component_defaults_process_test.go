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
				for _, name := range []string{"NullFreeDto", "NullEmptyDto", "NullOpenDto"} {
					text := string(read(t, p.Output+"/org/example/catalog/models/"+name+".kt"))
					if !strings.Contains(text, "public fun `"+name+" default`(): "+name+" =") || !strings.Contains(text, `("null")`) || !strings.Contains(text, "private fun `openapi-kt component default ") {
						t.Fatal("nullable raw factory lost its alias/helper", name, text)
					}
				}

				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, args...), 0)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("no-op generation or readonly check changed bytes or mtimes")
				}
				for _, tc := range []struct{ original, invalid, code, pointer string }{
					{`"default": {"kept": null, "z": {"$ref": "data"}}`, `"default": [1]`, "E_SCHEMA_INVALID", "/components/schemas/WrappedOpen/default"},
					{`"MaybeOpenTarget": {"type": ["object", "null"], "additionalProperties": true}`, `"MaybeOpenTarget": {"type": "object", "additionalProperties": true}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/WrappedNilOpen/default"},
				} {
					bad := bytes.Replace(document, []byte(tc.original), []byte(tc.invalid), 1)
					if bytes.Equal(bad, document) {
						t.Fatal("raw wrapper fixture case missing", tc.original)
					}
					put(t, p.Input, bad)
					for _, command := range [][]string{args, append(append([]string{}, args...), "--check")} {
						failure := fails(t, invoke(t, p.Root, command...), 1, tc.code)
						if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
							t.Fatal("invalid raw wrapper default did not preserve output", failure)
						}
					}
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
				badNull := bytes.Replace(document, []byte(`"type": "object", "nullable": true, "default": null`), []byte(`"type": "object", "nullable": false, "default": null`), 1)
				if bytes.Equal(badNull, document) {
					t.Fatal("nullable raw annotation missing from fixture")
				}
				put(t, p.Input, badNull)
				for _, command := range [][]string{args, append(append([]string{}, args...), "--check")} {
					failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_UNSUPPORTED")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/NullFree/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("non-nullable raw null default did not preserve output", failure)
					}
				}
			})
		}
	}
}
