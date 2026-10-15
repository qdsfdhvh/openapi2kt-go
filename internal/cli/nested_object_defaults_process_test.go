package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedObjectDocument(defaultValue, property string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"cfg":{"type":"object","properties":` + property + `,"default":` + defaultValue + `}}}}}}`)
}

func nestedObjectArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestNestedObjectDefaultsNativeCoreParity(t *testing.T) {
	property := `{"name":{"type":"string"},"age":{"type":"integer"},"status":{"type":"string","enum":["on","off"]}}`
	defaultValue := `{"name":"outer","age":9,"status":"on"}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := nestedObjectDocument(defaultValue, property)
				put(t, p.Input, document)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				args := nestedObjectArgs(p.Input, p.Output, checked, style)
				actual := result(t, invoke(t, p.Root, args...), 0)
				if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
					t.Fatal("core/native report mismatch", actual, report.Warnings)
				}
				for _, file := range report.Artifacts.Files {
					if string(read(t, p.Output+"/"+file.Path)) != file.Content {
						t.Fatal("native/core source mismatch", file.Path)
					}
					if !strings.Contains(file.Content, "`openapi-kt model default org example catalog models Holder Cfg`(") {
						t.Fatal("missing qualified object default helper", file.Path)
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

func TestNestedObjectDefaultsPreserveOutputOnFailure(t *testing.T) {
	p := newProject(t)
	good := nestedObjectDocument(`{"name":"outer"}`, `{"name":{"type":"string"}}`)
	put(t, p.Input, good)
	args := nestedObjectArgs(p.Input, p.Output, true, core.DataClass)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, p.Output)
	// A missing required property is a generation failure, not a partial write.
	bad := nestedObjectDocument(`{}`, `{"name":{"type":"string"}}`)
	put(t, p.Input, []byte(strings.Replace(string(bad), `"properties":{"name":{"type":"string"}}`, `"required":["name"],"properties":{"name":{"type":"string"}}`, 1)))
	fails(t, invoke(t, p.Root, args...), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed generation changed existing output")
	}
}

func TestNestedObjectDefaultsOptimizationRelocation(t *testing.T) {
	for _, options := range [][]string{
		{"--deduplicate-models"},
		{"--merge-identical-models"},
		{"--hoist-shared-shapes", "--deduplicate-models", "--merge-identical-models"},
	} {
		p := newProject(t)
		document := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"operationId":"a","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"cfg":{"type":"object","properties":{"x":{"type":"string"}},"default":{"x":"v"}}}}}}}}}},"/b":{"get":{"operationId":"b","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"cfg":{"type":"object","properties":{"x":{"type":"string"}},"default":{"x":"v"}}}}}}}}}}}}`)
		put(t, p.Input, document)
		args := append([]string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.opt", "--diagnostic-format=json"}, options...)
		result(t, invoke(t, p.Root, args...), 0)
		for name := range snapshot(t, p.Output) {
			if !strings.HasSuffix(name, ".kt") {
				continue
			}
			content := string(read(t, p.Output+"/"+name))
			// Every call site must bind a helper declared in the same file.
			for _, call := range nestedObjectHelperCalls(content) {
				if !strings.Contains(content, "private fun "+call+"(") {
					t.Fatalf("helper %s not declared in %s with %v", call, name, options)
				}
			}
		}
	}
}

// nestedObjectHelperCalls returns the backticked helper names used by object
// default initializers in one generated source file.
func nestedObjectHelperCalls(content string) []string {
	const prefix = "`openapi-kt model default "
	var names []string
	for idx := 0; ; {
		start := strings.Index(content[idx:], "= "+prefix)
		if start < 0 {
			return names
		}
		start += idx + len("= "+prefix)
		end := strings.Index(content[start:], "`(")
		if end < 0 {
			return names
		}
		names = append(names, prefix+content[start:start+end]+"`")
		idx = start + end + 2
	}
}
