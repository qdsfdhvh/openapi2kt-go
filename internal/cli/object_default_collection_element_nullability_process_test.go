package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectDefaultCollectionElementNullableNativeParity(t *testing.T) {
	valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Day":{"type":"string","format":"date"},"MaybeDay":{"type":["string","null"],"format":"date"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day","nullable":true}},"Values":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/MaybeDay","nullable":false}},"Holder":{"properties":{"v":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"},"values":{"$ref":"#/components/schemas/Values"}},"default":{"rows":[null,"2024-02-29"],"values":{"a/~":"2024-02-29"}}}}}}}}`)
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
			for _, file := range report.Artifacts.Files {
				if string(read(t, p.Output+"/"+file.Path)) != file.Content {
					t.Fatal("native/core parity", file.Path)
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
				t.Fatal("no-op changed output")
			}
			put(t, p.Input, []byte(strings.Replace(string(valid), `"values":{"a/~":"2024-02-29"}`, `"values":{"a/~":null}`, 1)))
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/values/a~1~0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost element pointer/old output", failure.Diagnostics)
				}
			}
		}
	}
}
