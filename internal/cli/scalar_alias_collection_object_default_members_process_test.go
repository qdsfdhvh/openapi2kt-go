package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestScalarAliasCollectionObjectDefaultNativeParity(t *testing.T) {
	valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Day":{"type":["string","null"],"format":"date"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Day","nullable":false}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Rows":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Alias","nullable":true}},"Holder":{"properties":{"v":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":{"a/~":"2024-02-29","nil":null}}}}}}}}`)
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
			put(t, p.Input, []byte(strings.Replace(string(valid), `"a/~":"2024-02-29"`, `"a/~":"2023-02-29"`, 1)))
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/rows/a~1~0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost element Pointer or old output", failure.Diagnostics)
				}
			}
		}
	}
}
