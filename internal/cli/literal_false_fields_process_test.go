package cli_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestLiteralFalseFieldsNativeParity(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/literal-false-fields.json"))
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			put(t, p.Input, valid)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Artifacts.Files {
				if string(read(t, p.Output+"/"+f.Path)) != f.Content {
					t.Fatal("native/core parity", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
			if checked {
				validate = append(validate, "--validation")
			}
			commands := [][]string{args, append(append([]string{}, args...), "--check"), validate}
			for _, command := range commands {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("readonly/noop changed output")
			}
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings", "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("false root warning policy changed output")
			}
			var bad map[string]any
			if err := json.Unmarshal(valid, &bad); err != nil {
				t.Fatal(err)
			}
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Optional"].(map[string]any)["properties"] = map[string]any{"blocked": map[string]any{"type": 12}}
			data, err := json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, data)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				pointer := "/components/schemas/Optional/properties/blocked"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost original source or existing output", failure.Diagnostics)
				}
			}
			nested := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"nested":{"properties":{"value":false}}}}}}}`)
			put(t, p.Input, nested)
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				pointer := "/components/schemas/Holder/properties/nested/properties/value"
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost excludedsource/oldoutput", failure.Diagnostics)
				}
			}
		}
	}
}

func TestLiteralFalseFieldsCumulativeBudgetNativePreservesOutput(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"blocked":false}}}}}`))
	args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, false, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	schemas := map[string]any{}
	for i := 0; i < 8; i++ {
		properties := map[string]any{}
		for j := 0; j < 240; j++ {
			properties[fmt.Sprintf("p%03d", j)] = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ZBlocked"}}
		}
		schemas[fmt.Sprintf("H%02d", i)] = map[string]any{"properties": properties}
	}
	schemas["ZBlocked"] = false
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, input)
	commands := [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--diagnostic-format=json"}}
	for _, command := range commands {
		failure := fails(t, invoke(t, p.Root, command...), 1, "E_MODEL_LIMIT")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/H07" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("false cumulativebudgetlostPointer/oldoutput", failure.Diagnostics)
		}
	}
}
