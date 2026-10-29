package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedObjectCollectionDocument() []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"a"},{"name":"b","age":3}]}}}}}}`)
}

func referencedObjectCollectionArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-name-suffix=Dto", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestReferencedObjectCollectionDefaultsNativeCoreParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := referencedObjectCollectionDocument()
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := referencedObjectCollectionArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				helperSeen := false
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					if strings.HasSuffix(file.Path, "/HolderDto.kt") && strings.Contains(file.Content, "`openapi-kt model default List org example catalog models ItemDto`(") {
						helperSeen = true
					}
				}
				if !helperSeen {
					t.Fatal("missing qualified list-of-referenced-DTO helper")
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

func TestReferencedObjectCollectionDefaultsPreserveOutputOnFailure(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, referencedObjectCollectionDocument())
	args := referencedObjectCollectionArgs(p.Input, p.Output, true, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	// The referenced DTO target validates its independent own annotation.
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":false}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"x"}]}}}}}}`))
	fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed generation changed existing output")
	}
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":"t"}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"x"}]}}}}}}`))
	result(t, invoke(t, p.Root, args...), 0)
	result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
}

func TestReferencedObjectCollectionDefaultsOptimizationRelocation(t *testing.T) {
	p := newProject(t)
	document := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","properties":{"x":{"type":"string"}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Rows","default":[{"x":"v"}]}}}}}}`)
	put(t, p.Input, document)
	args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.opt", "--diagnostic-format=json", "--merge-identical-models"}
	result(t, invoke(t, p.Root, args...), 0)
	for name := range snapshot(t, p.Output) {
		if !strings.HasSuffix(name, ".kt") {
			continue
		}
		content := string(read(t, p.Output+"/"+name))
		for _, prefix := range []string{"= `openapi-kt default tree ", "= `openapi-kt model default "} {
			for _, call := range strings.Split(content, prefix) {
				if call == content {
					continue
				}
				end := strings.Index(call, "`(")
				if end < 0 {
					continue
				}
				helper := strings.TrimPrefix(prefix, "= ") + call[:end] + "`"
				if !strings.Contains(content, "private fun "+helper+"(") {
					t.Fatalf("helper %s not declared in %s", helper, name)
				}
			}
		}
	}
}
