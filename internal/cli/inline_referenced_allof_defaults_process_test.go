package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func inlineReferencedAllOfDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","enum":[1,2]}}}]}}},"Hop":{"properties":{"node":{"$ref":"#/components/schemas/Owner/properties/node"}}},"Alias":{"$ref":"#/components/schemas/Hop/properties/node"},"Inline":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":` + value + `}}}}}}`)
}

func TestInlineReferencedAllOfNativeParityAndPreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			doc := inlineReferencedAllOfDocument(`[{"name":"a","n":2}]`)
			put(t, p.Input, doc)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: doc, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			args := referencedObjectCollectionArgs(p.Input, p.Output, checked, style)
			actual := result(t, invoke(t, p.Root, args...), 0)
			if len(actual.Files) != len(report.Artifacts.Files) || len(actual.Warnings) != len(report.Warnings) {
				t.Fatal("native/core reports differ")
			}
			seen := false
			for _, file := range report.Artifacts.Files {
				if string(read(t, p.Output+"/"+file.Path)) != file.Content {
					t.Fatal("native/core source differs", file.Path)
				}
				seen = seen || strings.Contains(file.Content, "private fun `openapi-kt default tree List Alias org example catalog models OwnerDto Node`")
			}
			if !seen {
				t.Fatal("lost canonical actual lexical element helper")
			}
			before := snapshot(t, p.Output)
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("check modified outputs")
			}
			for _, tc := range []struct{ value, code, tail string }{
				{`[{}]`, "E_SCHEMA_INVALID", "/0"},
				{`[{"name":"a","n":3}]`, "E_SCHEMA_INVALID", "/0/n"},
				{`[{"z/~":1,"a":false}]`, "E_SCHEMA_INVALID", "/0/z~1~0"},
				{`[{"name":"` + strings.Repeat("x", 32767) + `"}]`, "E_MODEL_LIMIT", "/0/name"},
			} {
				put(t, p.Input, inlineReferencedAllOfDocument(tc.value))
				failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
				if len(failure.Files) != 0 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Inline/properties/v/default"+tc.tail {
					t.Fatal("lost original annotation Pointer", failure)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failure changed existing bytes/modes/mtimes")
				}
			}
		}
	}
}
