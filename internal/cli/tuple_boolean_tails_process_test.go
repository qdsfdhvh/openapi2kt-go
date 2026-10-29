package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestTupleBooleanTailsNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-boolean-tails.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			put(t, p.Input, input)
			args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			result(t, invoke(t, p.Root, args...), 0)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, filepath.FromSlash(f.Path)))) != f.Content {
					t.Fatal("Boolean tail core/native mismatch", f.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning", "W_TUPLE_APPROXIMATION", "--check"}} {
				result(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Boolean tail check/no-op changed output")
				}
			}
			warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("warning denial changed old output")
			}
			// A true tail must not hide the invalid original prefix that its
			// raw final element carrier no longer exposes.
			bad := bytes.Replace(input, []byte(`"prefixItems": [`), []byte(`"prefixItems": [{"type":"string","unknown":true},`), 1)
			bad = bytes.Replace(bad, []byte(`"items": false`), []byte(`"items": true`), 1)
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				failure := fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_UNSUPPORTED")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Closed/prefixItems/0/unknown" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Boolean tail failure lost source or old output", failure)
				}
			}
		}
	}
}
