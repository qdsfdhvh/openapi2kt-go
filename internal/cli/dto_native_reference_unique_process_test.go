package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTONativeReferenceUniqueNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/dto-native-reference-unique.json"))
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
				t.Fatal("native/core DTONativeReferenceunique parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		for _, args := range [][]string{append(append([]string{}, flags...), "--check"), flags} {
			result(t, p.generate(t, args...), 0)
		}
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed DTONativeReferenceunique output")
		}

		for _, badEnum := range []bool{false, true} {
			var doc map[string]any
			if err := json.Unmarshal(input, &doc); err != nil {
				t.Fatal(err)
			}
			catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
			field := catalog["Item"].(map[string]any)["properties"].(map[string]any)["refInt"].(map[string]any)
			pointer := "/components/schemas/Item/properties/refInt/default"
			if badEnum {
				catalog["RefInt"].(map[string]any)["minimum"] = "bad"
				pointer = "/components/schemas/RefInt/minimum"
			} else {
				field["default"] = -1
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

		for _, depthFailure := range []bool{false, true} {
			field := map[string]any{"type": "number", "format": "float", "enum": []any{0, 1}}
			pointer := "/components/schemas/Holder/properties/values/default/1"
			expectedCode := "E_SCHEMA_INVALID"
			values := map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"$ref": "#/components/schemas/DefaultItem"}}
			if depthFailure {
				field = map[string]any{"type": "integer", "enum": []any{0, 1}}
				for range 16 {
					field = map[string]any{"type": "array", "items": field}
				}
				pointer = "/components/schemas/Holder/properties/values"
				expectedCode = "E_SCHEMA_DEPTH"
			} else {
				field = map[string]any{"type": "array", "items": field, "default": []any{0}}
				values["default"] = []any{map[string]any{"id": "a"}, map[string]any{"id": "a", "numbers": []any{0}}}
			}
			doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"DefaultItem": map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}, "numbers": field}, "required": []string{"id"}}, "Holder": map[string]any{"properties": map[string]any{"values": values}}}}}
			bytes, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bytes)
			failure := fails(t, p.generate(t, flags...), 1, expectedCode)
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("collection getter failure lost Pointer/old output", failure)
			}
		}
		bad := strings.Replace(string(input), `"uniqueItems": true`, `"uniqueItems": "bad"`, 1)
		put(t, p.Input, []byte(bad))
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/values/uniqueItems" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("DTONativeReferenceunique failure lost originalPointer/oldoutput", failure)
		}
	}
}
