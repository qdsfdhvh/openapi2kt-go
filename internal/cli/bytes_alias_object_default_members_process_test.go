package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBytesAliasObjectDefaultMembersNativeParity(t *testing.T) {
	valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blob":{"type":["string","null"],"format":"binary"},"Alias":{"$ref":"#/components/schemas/Blob","nullable":false},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":[-128,0,127]}}}}}}}`)
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
				t.Fatal("no-op changed output")
			}
			put(t, p.Input, []byte(strings.Replace(string(valid), "[-128,0,127]", "[-128,128,127]", 1)))
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/member/1" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost byte pointer/output", failure.Diagnostics)
				}
			}
		}
	}
}
