package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumAliasObjectDefaultMembersNativeParity(t *testing.T) {
	valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"number","format":"float","enum":[1,1.5]},"Alias":{"$ref":"#/components/schemas/Code"},"Bridge":{"$ref":"#/components/schemas/Alias"},"Holder":{"properties":{"v":{"properties":{"a/~":{"$ref":"#/components/schemas/Bridge"}},"default":{"a/~":1.5}}}}}}}`)
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
			// Same mapped Float, different exact source number: always reject.
			put(t, p.Input, []byte(strings.Replace(string(valid), `"default":{"a/~":1.5}`, `"default":{"a/~":1.5000000000000001}`, 1)))
			for _, command := range commands {
				failure := fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/a~1~0" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("lost exact membership pointer/old output", failure.Diagnostics)
				}
			}
		}
	}
}
