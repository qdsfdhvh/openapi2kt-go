package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedAllOfDefaultDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":` + value + `},"bridged":{"$ref":"#/components/schemas/Bridge","default":{"label":"bridge","n":2}}}},"Alias":{"$ref":"#/components/schemas/Value"},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/hop"},"Value":{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer","enum":[1,2]}}}]}}}}`)
}

func TestReferencedAllOfDefaultsNativeCoreParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := referencedAllOfDefaultDocument(`{"n":2,"label":"x"}`)
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := referencedObjectArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				helperSeen := false
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					helperSeen = helperSeen || strings.HasSuffix(file.Path, "/HolderDto.kt") && strings.Contains(file.Content, "`openapi-kt model default org example catalog models ValueDto`(")
				}
				if !helperSeen {
					t.Fatal("missing actual allOf DTO helper")
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed output bytes/modes/mtimes")
				}
				for _, tc := range []struct{ value, code, tail string }{
					{`{"label":false}`, "E_SCHEMA_INVALID", "/label"},
					{`{"label":"x","n":0}`, "E_SCHEMA_INVALID", "/n"},
					{`{}`, "E_SCHEMA_INVALID", ""},
					{`{"label":"` + strings.Repeat("x", 32767) + `"}`, "E_MODEL_LIMIT", "/label"},
				} {
					put(t, p.Input, referencedAllOfDefaultDocument(tc.value))
					report := fails(t, invoke(t, p.Root, args...), 1, tc.code)
					if len(report.Diagnostics) != 1 || report.Diagnostics[0].Pointer == nil || *report.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value/default"+tc.tail {
						t.Fatal("lost consuming default Pointer", report.Diagnostics)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("failure changed existing output")
					}
				}
			})
		}
	}
}
