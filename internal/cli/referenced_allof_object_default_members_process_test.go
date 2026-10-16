package cli_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestReferencedAllOfObjectDefaultMembersNativeParityAndPreservation(t *testing.T) {
	const target = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":2}}}]}`
	for _, reference := range []string{"Target", "Outer", "Owner/properties/node", "InternalAlias", "Hops/properties/first", "Bridge", "Rows", "RowsAlias", "RowsBridge", "RowsOuter", "RowsHops/properties/first", "Owner/properties/rows", "InternalRowsAlias"} {
		for _, collection := range []bool{false, true} {
			collectionMember := reference == "Rows" || reference == "RowsAlias" || reference == "RowsBridge" || reference == "RowsOuter" || reference == "RowsHops/properties/first" || reference == "Owner/properties/rows" || reference == "InternalRowsAlias"
			if collectionMember && collection {
				continue
			}
			member := `{"$ref":"#/components/schemas/` + reference + `"}`
			valid, invalid, tail := `{"member":{"name":"x"}}`, `{"member":{"name":false}}`, "/name"
			if collection || collectionMember {
				if collection {
					member = `{"type":"array","items":{"type":"object","additionalProperties":` + member + `}}`
				}
				valid, invalid, tail = `{"member":[{"k":{"name":"x"}}]}`, `{"member":[{"k":{"name":false}}]}`, "/0/k/name"
			}
			document := func(value string) []byte {
				return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":` + target + `,"Owner":{"properties":{"node":` + target + `,"rows":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}}}},"InternalRowsAlias":{"$ref":"#/components/schemas/Owner/properties/rows"},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"},"second":{"$ref":"#/components/schemas/InternalAlias"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},"InternalAlias":{"$ref":"#/components/schemas/Owner/properties/node"},"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"$ref":"#/components/schemas/Alias"},"Rows":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}},"RowsAlias":{"$ref":"#/components/schemas/Rows"},"RowsHops":{"properties":{"first":{"$ref":"#/components/schemas/RowsHops/properties/second"},"second":{"$ref":"#/components/schemas/Rows"}}},"RowsBridge":{"$ref":"#/components/schemas/RowsHops/properties/first"},"RowsOuter":{"$ref":"#/components/schemas/RowsBridge"},"Holder":{"properties":{"cfg":{"properties":{"member":` + member + `},"default":` + value + `}}}}}}`)
			}
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					p := newProject(t)
					input := document(valid)
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
					put(t, p.Input, document(invalid))
					failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
					if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/cfg/default/member"+tail {
						t.Fatal(failure)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed output")
					}
				}
			}
		}
	}
}
