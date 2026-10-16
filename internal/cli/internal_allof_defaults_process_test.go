package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func internalAllOfDefaultDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":` + value + `}}},"Owner":{"properties":{"node":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer","enum":[1,2]}}}]}}},"Hops":{"properties":{"via":{"$ref":"#/components/schemas/Owner/properties/node"}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/via"}}}}`)
}

func TestInternalAllOfDefaultsNativeParityAndPreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				doc := internalAllOfDefaultDocument(`{"n":2,"label":"x"}`)
				put(t, p.Input, doc)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: doc, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := referencedObjectArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Files) != len(report.Artifacts.Files) || len(actual.Warnings) != len(report.Warnings) {
					t.Fatal("core/native report mismatch")
				}
				seen := false
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("core/native source mismatch", file.Path)
					}
					seen = seen || strings.Contains(file.Content, "`openapi-kt model default org example catalog models OwnerDto Node`(")
				}
				if !seen {
					t.Fatal("lost lexical DTO helper")
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed output bytes/modes/mtimes")
				}
				for _, tc := range []struct{ value, code, tail string }{
					{`{"label":false}`, "E_SCHEMA_INVALID", "/label"},
					{`{"label":"x","n":3}`, "E_SCHEMA_INVALID", "/n"},
					{`{}`, "E_SCHEMA_INVALID", ""},
					{`{"label":"` + strings.Repeat("x", 32767) + `"}`, "E_MODEL_LIMIT", "/label"},
				} {
					put(t, p.Input, internalAllOfDefaultDocument(tc.value))
					failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default"+tc.tail {
						t.Fatal("lost default Pointer", failure.Diagnostics)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed existing output")
					}
				}
			})
		}
	}
}
