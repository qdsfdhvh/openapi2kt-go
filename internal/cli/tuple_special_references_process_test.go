package cli_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleSpecialReferencesNative(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-special-references.json"))
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
					t.Fatal("special reference core/native drift", file.Path)
				}
			}
			before := snapshot(t, p.Output)
			for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning", "W_TUPLE_APPROXIMATION", "--allow-warning", "W_JSON_ELEMENT_APPROXIMATION", "--check"}} {
				result(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 0)
			}
			warningDenied(t, invoke(t, p.Root, append(append([]string{}, args...), "--deny-warnings")...))
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("read-only/no-op special reference tuple generation changed existing files")
			}
			bad := bytes.Replace(input, []byte(`"x-native-type": "bigint"`), []byte(`"unknownTupleRole": true, "x-native-type": "bigint"`), 1)
			if bytes.Equal(input, bad) {
				t.Fatal("fixture missing original special reference mutation")
			}
			put(t, p.Input, bad)
			for _, extra := range [][]string{nil, {"--check"}} {
				fails(t, invoke(t, p.Root, append(append([]string{}, args...), extra...)...), 1, "E_SCHEMA_UNSUPPORTED")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid original special reference replaced existing output")
				}
			}
			if checked && style == core.PlainClass {
				parts := []string{`"Value":{"type":"integer","enum":[1,2]}`}
				for i := 0; i < 1400; i++ {
					parts = append(parts, fmt.Sprintf(`"A%04d":{"$ref":"#/components/schemas/Value"},"Rows%04d":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/A%04d"}]}`, i, i, i))
				}
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+strings.Join(parts, ",")+`}}}`))
				fails(t, invoke(t, p.Root, args...), 1, "E_MODEL_LIMIT")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("over-budget named alias capture replaced existing output")
				}
			}
		}
	}
}
