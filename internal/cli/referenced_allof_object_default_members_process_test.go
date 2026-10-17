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

// A referenced collection whose inline element tree ends in a bare ordinary
// allOf constructor borrows that retained element. Native generation, readonly
// check and failure preservation match the core plan.
func TestReferencedCollectionBareAllOfMemberNativeParityAndPreservation(t *testing.T) {
	const leaf = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":2}}}]}`
	for _, tc := range []struct{ name, collection, valid, invalid, tail string }{
		{"list", `{"type":"array","items":` + leaf + `}`, `{"member":[{"name":"x"}]}`, `{"member":[{"name":false}]}`, "/0/name"},
		{"map", `{"type":"object","additionalProperties":` + leaf + `}`, `{"member":{"k":{"name":"x"}}}`, `{"member":{"k":{"name":false}}}`, "/k/name"},
		{"mixed", `{"type":"array","items":{"type":"object","additionalProperties":` + leaf + `}}`, `{"member":[{"k":{"name":"x"}}]}`, `{"member":[{"k":{"name":false}}]}`, "/0/k/name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := func(value string) []byte {
				return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":` + tc.collection + `,"Holder":{"properties":{"cfg":{"properties":{"member":{"$ref":"#/components/schemas/Rows"}},"default":` + value + `}}}}}}`)
			}
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					p := newProject(t)
					input := document(tc.valid)
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
					put(t, p.Input, document(tc.invalid))
					failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
					if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/cfg/default/member"+tc.tail {
						t.Fatal(failure)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed output")
					}
				}
			}
		})
	}
}

// An internal allOf element terminal inside a named collection is admitted
// through the shared resolver seam; native generation, readonly check and
// failure preservation match the core plan.
func TestReferencedCollectionInternalAllOfElementTerminalParity(t *testing.T) {
	const target = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":2}}}]}`
	for _, tc := range []struct {
		name, collection, valid, invalid, tail string
	}{
		{"list", `{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/node"}}`, `{"member":[{"name":"x"}]}`, `{"member":[{"name":false}]}`, "/0/name"},
		{"map", `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Owner/properties/node"}}`, `{"member":{"k":{"name":"x"}}}`, `{"member":{"k":{"name":false}}}`, "/k/name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := func(value string) []byte {
				return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":` + target + `}},"Rows":` + tc.collection + `,"Alias":{"$ref":"#/components/schemas/Rows"},"Holder":{"properties":{"cfg":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":` + value + `}}}}}}`)
			}
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					p := newProject(t)
					input := document(tc.valid)
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
					put(t, p.Input, document(tc.invalid))
					failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
					if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/cfg/default/member"+tc.tail {
						t.Fatal(failure)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed output")
					}
				}
			}
		})
	}
}

// An object-default property may reference a collection whose element resolves
// to a plain named ordinary DTO (directly or through a finite alias chain or an
// inline container). Native generation, readonly check and failure preservation
// match the core plan.
func TestReferencedCollectionPlainDTOElementsParity(t *testing.T) {
	const item = `{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7}}}`
	for _, tc := range []struct{ name, member, valid, invalid, tail string }{
		{"named list", `{"$ref":"#/components/schemas/Rows"}`, `{"member":[{"name":"a"}]}`, `{"member":[{"name":false}]}`, "/member/0/name"},
		{"named map", `{"$ref":"#/components/schemas/MapRows"}`, `{"member":{"k":{"name":"a"}}}`, `{"member":{"k":{}}}`, "/member/k"},
		{"alias-chain element", `{"$ref":"#/components/schemas/AliasRows"}`, `{"member":[{"name":"a"}]}`, `{"member":[{}]}`, "/member/0"},
		{"inline list", `{"type":"array","items":{"$ref":"#/components/schemas/Item"}}`, `{"member":[{"name":"a"}]}`, `{"member":[{"z":1}]}`, "/member/0/z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := func(value string) []byte {
				return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":` + item + `,"ItemAlias":{"$ref":"#/components/schemas/Item"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"AliasRows":{"type":"array","items":{"$ref":"#/components/schemas/ItemAlias"}},"MapRows":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"v":{"properties":{"member":` + tc.member + `},"default":` + value + `}}}}}}`)
			}
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					p := newProject(t)
					input := document(tc.valid)
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
					put(t, p.Input, document(tc.invalid))
					failure := fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
					if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default"+tc.tail {
						t.Fatal(failure)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed output")
					}
				}
			}
		})
	}
}
