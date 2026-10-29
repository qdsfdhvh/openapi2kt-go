package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestTupleScalarConstNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-scalar-const.json"))
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
			for _, file := range report.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, filepath.FromSlash(file.Path)))) != file.Content {
					t.Fatal("const core/native drift", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning", "W_TUPLE_APPROXIMATION", "--allow-warning", "W_CONST_SCALAR_APPROXIMATION", "--check"}} {
				result(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 0)
			}
			warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("read-only/no-op const tuple generation changed existing files")
			}
			bad := bytes.Replace(input, []byte(`"const": "fixed"`), []byte(`"unknownTupleRole": true, "const": "fixed"`), 1)
			if bytes.Equal(input, bad) {
				t.Fatal("fixture missing original const mutation")
			}
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_UNSUPPORTED")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid original null vocabulary replaced existing output")
				}
			}
		}
	}
}
