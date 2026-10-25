package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestTupleRootDefaultsNativeParityAndPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-root-defaults.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, valid)
		args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, style)
		result(t, invoke(t, p.Root, args...), 0)
		want, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range want.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("tuple root native/core parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--validation", "--diagnostic-format=json"}}
		for _, cmd := range commands {
			result(t, invoke(t, p.Root, cmd...), 0)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed output")
		}
		for _, bad := range []string{"outer", "original-own", "alias"} {
			var d map[string]any
			decoder := json.NewDecoder(bytes.NewReader(valid))
			decoder.UseNumber()
			if err := decoder.Decode(&d); err != nil {
				t.Fatal(err)
			}
			components := d["components"].(map[string]any)["schemas"].(map[string]any)
			pointer := "/components/schemas/Rows/default/0"
			switch bad {
			case "outer":
				components["Rows"].(map[string]any)["default"] = []any{true}
			case "original-own":
				components["Longs"].(map[string]any)["default"] = []any{true}
				pointer = "/components/schemas/Longs/default/0"
			case "alias":
				components["Wrapped"].(map[string]any)["default"] = []any{true}
				pointer = "/components/schemas/Wrapped/default/0"
			}
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, cmd := range commands {
				code := "E_SCHEMA_INVALID"
				failure := fails(t, invoke(t, p.Root, cmd...), 1, code)
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Tuple default failure lost original Pointer or output", bad, failure)
				}
			}
		}
	}
}
