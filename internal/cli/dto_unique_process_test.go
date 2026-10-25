package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOUniqueNativeParityAndPreservation(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/dto-unique.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, input)
		flags := []string{"--validation", "--model-name-suffix", "Dto", "--model-style", string(style)}
		result(t, p.generate(t, flags...), 0)
		expected, e := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true}})
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range expected.Artifacts.Files {
			if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
				t.Fatal("native/core DTOunique parity", f.Path)
			}
		}
		before := snapshot(t, p.Output)
		for _, args := range [][]string{append(append([]string{}, flags...), "--check"), flags} {
			result(t, p.generate(t, args...), 0)
		}
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("readonly/noop changed DTOunique output")
		}
		bad := strings.Replace(string(input), `"uniqueItems": true`, `"uniqueItems": "bad"`, 1)
		put(t, p.Input, []byte(bad))
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/values/uniqueItems" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("DTOunique failure lost originalPointer/oldoutput", failure)
		}
	}
}
