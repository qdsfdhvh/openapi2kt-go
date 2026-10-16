package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func objectDefaultReferencedPropertyDocument(defaultValue string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"Mode":{"type":"string","enum":["a","b"]},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Leaf"},"mode":{"$ref":"#/components/schemas/Mode"}},"default":` + defaultValue + `}}}}}}`)
}

func objectDefaultReferencedPropertyArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-name-suffix=Dto", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestObjectDefaultReferencedPropertiesNativeCoreParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := objectDefaultReferencedPropertyDocument(`{"leaf":{"z":"1"},"mode":"a"}`)
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					if strings.HasSuffix(file.Path, "/HolderDto.kt") && !strings.Contains(file.Content, `{\"leaf\":{\"z\":\"1\"},\"mode\":\"a\"}`) {
						t.Fatal("referenced property default JSON", file.Content)
					}
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

func TestObjectDefaultReferencedPropertiesPreserveOutputOnFailure(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, objectDefaultReferencedPropertyDocument(`{"leaf":{"z":"1"}}`))
	args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, objectDefaultReferencedPropertyDocument(`{"leaf":{}}`))
	fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed generation changed existing output")
	}
}
