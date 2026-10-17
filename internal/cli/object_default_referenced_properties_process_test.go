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

func objectDefaultPlainObjectAliasDocument(defaultValue string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":true},"Outer":{"$ref":"#/components/schemas/Alias","nullable":false},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Outer"}},"default":` + defaultValue + `}}}}}}`)
}

// A finite transparent chain to a plain object terminal keeps the original
// terminal's default and diagnostics. Native generation, readonly check and
// failure preservation match the core plan.
func TestObjectDefaultPlainObjectAliasNativeCoreParity(t *testing.T) {
	const definitions = `"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":true},"Outer":{"$ref":"#/components/schemas/Alias","nullable":false},`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				valid := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + definitions + `"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Outer"}},"default":{"leaf":{"z":"1"}}}}}}}}`)
				put(t, p.Input, valid)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed bytes/modes/mtimes")
				}
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+definitions+`"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Outer"}},"default":{"leaf":{}}}}}}}}`))
				fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failed generation changed existing output")
				}
			})
		}
	}
}

// An object-default property may reference a primitive union component. Native
// generation, readonly check and failure preservation match the core plan, and a
// non-member keeps the original member Pointer.
func TestObjectDefaultReferencedUnionPropertyParity(t *testing.T) {
	const definitions = `"V":{"oneOf":[{"type":"string"},{"type":"integer"}]},`
	document := func(member string) []byte {
		return []byte(`{"openapi":"3.1.0","components":{"schemas":{` + definitions + `"Holder":{"properties":{"v":{"type":"object","properties":{"u":{"$ref":"#/components/schemas/V"}},"default":{"u":` + member + `}}}}}}}`)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				valid := document(`"x"`)
				put(t, p.Input, valid)
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: valid, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed bytes/modes/mtimes")
				}
				put(t, p.Input, document(`true`))
				fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failed generation changed existing output")
				}
			})
		}
	}
}
