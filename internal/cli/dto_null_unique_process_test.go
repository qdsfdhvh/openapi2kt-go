package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTONullUniqueNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/dto-null-unique.json"))
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
				t.Fatal("native/core DTOnullunique parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		for _, args := range [][]string{append(append([]string{}, flags...), "--check"), flags} {
			result(t, p.generate(t, args...), 0)
		}
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed DTOnullunique output")
		}

		for _, badEnum := range []bool{false, true} {
			var doc map[string]any
			if err := json.Unmarshal(input, &doc); err != nil {
				t.Fatal(err)
			}
			catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
			field := catalog["Item"].(map[string]any)["properties"].(map[string]any)["nullEnum"].(map[string]any)
			pointer := "/components/schemas/Item/properties/nullEnum/default"
			if badEnum {
				field["enum"] = []any{nil, nil}
				pointer = "/components/schemas/Item/properties/nullEnum/enum/1"
			} else {
				field["default"] = "null"
			}
			bytes, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bytes)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("null vocabulary/default failure lost Pointer or old output", failure)
			}
		}

		bad := strings.Replace(string(input), `"uniqueItems": true`, `"uniqueItems": "bad"`, 1)
		put(t, p.Input, []byte(bad))
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/values/uniqueItems" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("DTOnullunique failure lost originalPointer/oldoutput", failure)
		}
	}
}
