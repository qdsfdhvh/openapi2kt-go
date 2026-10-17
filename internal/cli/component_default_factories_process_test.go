package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func componentDefaultDocument(value string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Simple":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":` + value + `}}}}`)
}

// Native generation, readonly check and failure preservation match the core
// plan for a component-level default factory.
func TestComponentDefaultFactoryParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := componentDefaultDocument(`{"z":"1"}`)
				put(t, p.Input, document)
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/SimpleDto.kt") && !strings.Contains(string(content), "public fun default(): SimpleDto =") {
						t.Fatal("factory missing from native output", content)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed bytes/modes/mtimes")
				}
				put(t, p.Input, componentDefaultDocument(`{}`))
				fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failed generation changed existing output")
				}
			})
		}
	}
}
