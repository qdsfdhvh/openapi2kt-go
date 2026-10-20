package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestScalarAliasObjectDefaultMembersNativeParity(t *testing.T) {
	valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"id":{"type":"string","format":"uuid","nullable":true}}},"Alias":{"$ref":"#/components/schemas/Source/properties/id","nullable":false},"Outer":{"$ref":"#/components/schemas/Alias"},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Outer"}},"default":{"member":"550e8400-e29b-41d4-a716-446655440000"}}}}}}}`)
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
			for _, value := range []string{"null", "1"} {
				put(t, p.Input, []byte(strings.Replace(string(valid), `"member":"550e8400-e29b-41d4-a716-446655440000"`, `"member":`+value, 1)))
				for _, command := range commands {
					failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/member" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("lost pointer/output", failure.Diagnostics)
					}
				}
			}
		}
	}
}
