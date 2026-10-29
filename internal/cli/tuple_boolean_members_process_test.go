package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestTupleBooleanMembersNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-boolean-members.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
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
					t.Fatal("Boolean positional tuple core/native divergence", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning", "W_TUPLE_APPROXIMATION", "--allow-warning", "W_ENUM_NATIVE_APPROXIMATION", "--check"}} {
				result(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("check/no-op changed bytes or mtimes")
			}
			warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			bad := bytes.Replace(input, []byte(`"prefixItems": [`), []byte(`"unknownTupleRole": true, "prefixItems": [`), 1)
			if bytes.Equal(bad, input) {
				t.Fatal("fixture did not exercise original source mutation")
			}
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_UNSUPPORTED")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid original enum source replaced old output")
				}
			}
			var doc map[string]any
			decoder := json.NewDecoder(bytes.NewReader(input))
			decoder.UseNumber()
			if err := decoder.Decode(&doc); err != nil {
				t.Fatal(err)
			}
			patch := doc["paths"].(map[string]any)["/patch"].(map[string]any)["patch"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			patch["properties"].(map[string]any)["optional"].(map[string]any)["default"] = []any{nil}
			bad, err = json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				failed := fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_INVALID")
				if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/optional/default/0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Patch suppression hid invalid original false-element annotation", failed)
				}
			}
		}
	}
}
