package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func rawJSONFieldDocument() []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"open":{"type":"object","additionalProperties":true,"default":{"a":1}},"free":{"default":[1,2,3]},"empty":{"type":"object","default":{"x":"y"}}}}}}}`)
}

func rawJSONFieldArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-name-suffix=Dto", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestRawJSONFieldDefaultsNativeCoreParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := rawJSONFieldDocument()
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := rawJSONFieldArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				helper := false
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					if strings.Contains(file.Content, "private fun `openapi-kt json default`(") && strings.Contains(file.Content, "private fun `openapi-kt object default`(") {
						helper = true
					}
				}
				if !helper {
					t.Fatal("missing raw JSON default helpers")
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed bytes/modes/mtimes")
				}
			})
		}
	}
}

func TestRawJSONFieldDefaultsPreserveOutputOnFailure(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, rawJSONFieldDocument())
	args := rawJSONFieldArgs(p.Input, p.Output, true, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"open":{"type":"object","additionalProperties":true,"default":[1]}}}}}}`))
	fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed generation changed existing output")
	}
}
