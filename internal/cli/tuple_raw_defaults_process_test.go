package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestRawTupleDefaultsNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-raw-defaults.json"))
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
					t.Fatal("raw tuple default core/native divergence", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning", "W_TUPLE_APPROXIMATION", "--check"}} {
				result(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("raw tuple default check/no-op changed bytes or mtimes")
			}
			warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			var doc map[string]any
			decoder := json.NewDecoder(bytes.NewReader(input))
			decoder.UseNumber()
			if err := decoder.Decode(&doc); err != nil {
				t.Fatal(err)
			}
			doc["components"].(map[string]any)["schemas"].(map[string]any)["Mixed"].(map[string]any)["default"] = map[string]any{"invalid": "array"}
			bad, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				failure := fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Mixed/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid raw annotation lost source/old output", failure)
				}
			}
		}
	}
}
