package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedAllOfCollectionDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"code":{"type":"integer","enum":[1,2]}}}]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":` + value + `}}}}}}`)
}

func TestReferencedAllOfCollectionNativeParityAndPreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			doc := referencedAllOfCollectionDocument(`[{"name":"a","code":2}]`)
			put(t, p.Input, doc)
			r := core.GenerationRequest{Document: doc, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}}
			report, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			args := referencedObjectCollectionArgs(p.Input, p.Output, checked, style)
			actual := result(t, invoke(t, p.Root, args...), 0)
			if len(actual.Files) != len(report.Artifacts.Files) || len(actual.Warnings) != len(report.Warnings) {
				t.Fatal("native/core report mismatch")
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, p.Output+"/"+file.Path)) != file.Content {
					t.Fatal("native/core source mismatch", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("check changed output")
			}
			for _, tc := range []struct{ value, code, tail string }{
				{`[{}]`, "E_SCHEMA_INVALID", "/0"},
				{`[{"name":"a","code":3}]`, "E_SCHEMA_INVALID", "/0/code"},
				{`[{"z/~":1,"a":false}]`, "E_SCHEMA_INVALID", "/0/z~1~0"},
				{`[{"name":"` + strings.Repeat("a", 32767) + `"}]`, "E_MODEL_LIMIT", "/0/name"},
			} {
				put(t, p.Input, referencedAllOfCollectionDocument(tc.value))
				failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
				if len(failure.Files) != 0 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default"+tc.tail {
					t.Fatal("lost original element Pointer", failure)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failure changed existing bytes/modes/mtimes")
				}
			}
		}
	}
}
