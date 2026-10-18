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

// Scalar and String-enum component factories keep native/core parity. A
// typealias cannot own members, so both forms emit a top-level function.
func TestComponentDefaultFactoryScalarParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"Name":{"type":"string","default":"x"},"Mode":{"type":"string","enum":["a","b"],"default":"a"}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/NameDto.kt") && !strings.Contains(string(content), "public fun `NameDto default`(): NameDto =") {
						t.Fatal("scalar factory missing", content)
					}
					if strings.Contains(file.Path, "/ModeDto.kt") && !strings.Contains(string(content), "public fun `ModeDto default`(): ModeDto =") {
						t.Fatal("enum factory missing", content)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("check changed bytes/modes/mtimes")
				}
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Name":{"type":"string","default":5}}}}`))
				fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("failed generation changed existing output")
				}
			})
		}
	}
}

// An allOf component factory keeps native/core parity and readonly stability.
func TestComponentDefaultFactoryAllOfParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"S":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}],"default":{"a":"x"}}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/SDto.kt") && !strings.Contains(string(content), "public fun default(): SDto =") {
						t.Fatal("allOf factory missing", content)
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

// A single-layer collection component factory keeps native/core parity.
func TestComponentDefaultFactoryCollectionParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"Names":{"type":"array","items":{"type":"string"},"default":["a"]},"Bags":{"type":"object","additionalProperties":{"type":"integer"},"default":{"k":1}}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/NamesDto.kt") && !strings.Contains(string(content), "public fun `NamesDto default`(): NamesDto =") {
						t.Fatal("List factory missing", content)
					}
					if strings.Contains(file.Path, "/BagsDto.kt") && !strings.Contains(string(content), "public fun `BagsDto default`(): BagsDto =") {
						t.Fatal("Map factory missing", content)
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

// A temporal component factory keeps native/core parity.
func TestComponentDefaultFactoryTemporalParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"When":{"type":"string","format":"date-time","default":"2024-01-02T03:04:05Z"},"Day":{"type":"string","format":"date","default":"2024-02-29"}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/WhenDto.kt") && !strings.Contains(string(content), "public fun `WhenDto default`(): WhenDto =") {
						t.Fatal("Instant factory missing", content)
					}
					if strings.Contains(file.Path, "/DayDto.kt") && !strings.Contains(string(content), "public fun `DayDto default`(): DayDto =") {
						t.Fatal("LocalDate factory missing", content)
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

// A native enum component factory keeps native/core parity.
func TestComponentDefaultFactoryNativeEnumParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","format":"int32","enum":[1,2],"default":1},"Flag":{"type":"boolean","enum":[true,false],"default":true}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/CodeDto.kt") && !strings.Contains(string(content), "public fun `CodeDto default`(): CodeDto =") {
						t.Fatal("native Int factory missing", content)
					}
					if strings.Contains(file.Path, "/FlagDto.kt") && !strings.Contains(string(content), "public fun `FlagDto default`(): FlagDto =") {
						t.Fatal("native Boolean factory missing", content)
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

// A nullable object component's null factory keeps native/core parity.
func TestComponentDefaultFactoryNullableParity(t *testing.T) {
	const document = `{"openapi":"3.1.0","components":{"schemas":{"S":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":null}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, []byte(document))
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: []byte(document), Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					content := read(t, p.Output+"/"+file.Path)
					if string(content) != file.Content {
						t.Fatal("native/core mismatch", file.Path)
					}
					if strings.Contains(file.Path, "/SDto.kt") && !strings.Contains(string(content), "public fun default(): SDto? = null") {
						t.Fatal("nullable factory missing", content)
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
