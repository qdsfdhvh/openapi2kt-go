package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func unionFieldDocument(defaultValue string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"},{"type":"number","format":"double"},{"type":"boolean"}],"default":` + defaultValue + `}}}}}}`)
}

func unionFieldArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-name-suffix=Dto", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestUnionFieldDefaultsNativeCoreParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := unionFieldDocument(`"x"`)
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := unionFieldArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				helperSeen := false
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					if strings.HasSuffix(file.Path, "/HolderDto.kt") && strings.Contains(file.Content, "`openapi-kt model default org example catalog models HolderDto V`(\"\\\"x\\\"\")") {
						helperSeen = true
					}
				}
				if !helperSeen {
					t.Fatal("missing qualified union default helper")
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

func TestUnionFieldDefaultsPreserveOutputOnFailure(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, unionFieldDocument(`"x"`))
	args := unionFieldArgs(p.Input, p.Output, true, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	// A token that matches no accepted branch must not partially generate.
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"boolean"}],"default":1}}}}}}`))
	fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed generation changed existing output")
	}
}

func TestUnionFieldDefaultsOptimizationRelocation(t *testing.T) {
	p := newProject(t)
	document := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Alpha":{"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"x"}}},"Zeta":{"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"x"}}},"Holder":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Alpha"}}}}}}`)
	put(t, p.Input, document)
	args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.opt", "--diagnostic-format=json", "--merge-identical-models"}
	result(t, invoke(t, p.Root, args...), 0)
	for name := range snapshot(t, p.Output) {
		if !strings.HasSuffix(name, ".kt") {
			continue
		}
		content := string(read(t, p.Output+"/"+name))
		for _, call := range strings.Split(content, "= `openapi-kt model default ") {
			if call == content {
				continue
			}
			end := strings.Index(call, "`(")
			if end < 0 {
				continue
			}
			helper := "`openapi-kt model default " + call[:end] + "`"
			if !strings.Contains(content, "private fun "+helper+"(") {
				t.Fatalf("helper %s not declared in %s", helper, name)
			}
		}
	}
}
