package cli_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestReferencedAllOfObjectDefaultMembersNativeParityAndPreservation(t *testing.T) {
	const target = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":2}}}]}`
	for _, reference := range []string{"Target", "Outer", "Owner/properties/node", "InternalAlias"} {
		document := func(value string) []byte {
			return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":` + target + `,"Owner":{"properties":{"node":` + target + `}},"InternalAlias":{"$ref":"#/components/schemas/Owner/properties/node"},"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"$ref":"#/components/schemas/Alias"},"Holder":{"properties":{"cfg":{"properties":{"member":{"$ref":"#/components/schemas/` + reference + `"}},"default":` + value + `}}}}}}`)
		}
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				p := newProject(t)
				input := document(`{"member":{"name":"x"}}`)
				put(t, p.Input, input)
				args := nestedObjectArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed output")
				}
				put(t, p.Input, document(`{"member":{"name":false}}`))
				failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/cfg/default/member/name" {
					t.Fatal(failure)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failure changed output")
				}
			}
		}
	}
}
