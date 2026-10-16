package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func allOfObjectDefaultDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"cfg":{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer"}}}],"default":` + value + `}}}}}}`)
}

func TestAllOfObjectDefaultsNativeParityAndFailurePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			p := newProject(t)
			document := allOfObjectDefaultDocument(`{"n":2,"name":"x"}`)
			put(t, p.Input, document)
			args := nestedObjectArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range report.Artifacts.Files {
				if string(read(t, p.Output+"/"+file.Path)) != file.Content {
					t.Fatal("native/core output mismatch", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("check changed bytes/modes/mtimes")
			}
			for _, tc := range []struct{ value, code, tail string }{
				{`{}`, "E_SCHEMA_INVALID", ""},
				{`{"name":"x","n":false}`, "E_SCHEMA_INVALID", "/n"},
				{`{"z/~":1,"name":"x"}`, "E_SCHEMA_INVALID", "/z~1~0"},
				{`{"name":"` + strings.Repeat("a", 32767) + `"}`, "E_MODEL_LIMIT", "/name"},
			} {
				put(t, p.Input, allOfObjectDefaultDocument(tc.value))
				failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
				if len(failure.Files) != 0 || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/cfg/default"+tc.tail {
					t.Fatal("wrong default diagnostic", failure)
				}
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failed default changed existing output")
				}
			}
		}
	}
}
