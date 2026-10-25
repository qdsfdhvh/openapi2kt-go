package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOConstUniqueNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/dto-const-unique.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, input)
		flags := []string{"--validation", "--model-name-suffix", "Dto", "--model-style", string(style)}
		result(t, p.generate(t, flags...), 0)
		expected, e := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range expected.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("native/core DTOconstunique parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		for _, args := range [][]string{append(append([]string{}, flags...), "--check"), flags} {
			result(t, p.generate(t, args...), 0)
		}
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed DTOconstunique output")
		}

		for _, badDefault := range []bool{false, true} {
			var doc map[string]any
			if err := json.Unmarshal(input, &doc); err != nil {
				t.Fatal(err)
			}
			catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
			pointer := "/components/schemas/Item/properties/state/default"
			if badDefault {
				values := catalog["Holder"].(map[string]any)["properties"].(map[string]any)["values"].(map[string]any)
				values["default"] = []any{map[string]any{"id": "a"}, map[string]any{"id": "a", "state": "ready", "kind": "fixed", "pinnedNumber": -0.0}}
				pointer = "/components/schemas/Holder/properties/values/default/1"
			} else {
				catalog["Item"].(map[string]any)["properties"].(map[string]any)["state"].(map[string]any)["default"] = "foreign"
			}
			bytes, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bytes)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("enum constructor-default failure lost Pointer or old output", failure)
			}
		}
		bad := strings.Replace(string(input), `"uniqueItems": true`, `"uniqueItems": "bad"`, 1)
		put(t, p.Input, []byte(bad))
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/values/uniqueItems" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("DTOconstunique failure lost originalPointer/oldoutput", failure)
		}
	}
}
