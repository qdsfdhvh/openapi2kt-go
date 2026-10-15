package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedNativeEnumDocument(field string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":` + field + `}}}}}`)
}

// Paths run outermost first. The annotation belongs only to the outer field.
func nestedNativeEnumField(path, leaf, value, rules string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == 'L' {
			leaf = `{"type":"array",` + rules + `"items":` + leaf + `}`
		} else {
			leaf = `{"type":"object",` + rules + `"additionalProperties":` + leaf + `}`
		}
	}
	return strings.TrimSuffix(leaf, "}") + `,"default":` + value + `}`
}

func nestedNativeEnumArgs(input, output string, checked bool, style core.ModelStyle) []string {
	args := []string{"generate", "--input", input, "--output", output, "--package-name=org.example.catalog", "--diagnostic-format=json", "--model-style=" + string(style)}
	if checked {
		args = append(args, "--validation")
	}
	return args
}

func TestNestedNativeEnumCollectionDefaultsNativeCoreParity(t *testing.T) {
	for _, leaf := range []struct{ name, schema, value string }{
		{"Int", `{"type":"integer","enum":[1,2]}`, `1`},
		{"Long", `{"type":"integer","format":"int64","enum":[-9223372036854775808,9223372036854775807]}`, `-9223372036854775808`},
		{"Float", `{"type":"number","format":"float","enum":[0.5,1]}`, `0.5`},
		{"Double", `{"type":"number","enum":[0.5,1]}`, `0.5`},
		{"Boolean", `{"type":"boolean","enum":[true,false]}`, `true`},
	} {
		for _, path := range []string{"LL", "LM", "ML", "MM", "LMLM"} {
			value := leaf.value
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == 'L' {
					value = `[` + value + `]`
				} else {
					value = `{"a/~":` + value + `}`
				}
			}
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/checked=%t", leaf.name, path, style, checked), func(t *testing.T) {
						p := newProject(t)
						document := nestedNativeEnumDocument(nestedNativeEnumField(path, leaf.schema, value, ""))
						put(t, p.Input, document)
						report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelStyle: style, Validation: checked}})
						if err != nil {
							t.Fatal(err)
						}
						args := nestedNativeEnumArgs(p.Input, p.Output, checked, style)
						actual := result(t, invoke(t, p.Root, args...), 0)
						if len(actual.Warnings) != len(report.Warnings) || len(actual.Files) != len(report.Artifacts.Files) {
							t.Fatal("core/native report mismatch", actual, report.Warnings)
						}
						for i, warning := range actual.Warnings {
							if warning.Code != report.Warnings[i].Code || !reflect.DeepEqual(warning.Pointer, report.Warnings[i].Pointer) {
								t.Fatal("core/native warning source mismatch", warning, report.Warnings[i])
							}
						}
						for _, file := range report.Artifacts.Files {
							if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content || !strings.Contains(file.Content, "kotlin."+leaf.name) {
								t.Fatal("actual source/type parity", file.Path)
							}
						}
						if checked && len(actual.Warnings) != 0 || !checked && (len(actual.Warnings) != 1 || actual.Warnings[0].Code != "W_ENUM_NATIVE_APPROXIMATION") {
							t.Fatal("native approximation policy", actual.Warnings)
						}
						before := snapshot(t, p.Output)
						result(t, invoke(t, p.Root, append(args, "--check")...), 0)
						if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
							t.Fatal("check changed bytes/modes/mtimes")
						}
					})
				}
			}
		}
	}
}

func nestedNativeEnumLockInput(t testing.TB, base string) string {
	t.Helper()
	input := filepath.Join(base, "input.json")
	field := `{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","nullable":true,"enum":[0.5,1]}},"default":{"z":{"z":null,"a":0.5},"a":{"/~":1}}}`
	op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{"description":"OK"}}}}`
	put(t, input, []byte(`{"openapi":"3.1.0","paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return input
}

func TestNestedNativeEnumCollectionDefaultsNativeLockReadonlyAndPreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			base := t.TempDir()
			input := nestedNativeEnumLockInput(t, base)
			output := filepath.Join(base, "out")
			args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
			if !checked {
				filtered := []string{}
				for _, arg := range args {
					if arg != "--validation" {
						filtered = append(filtered, arg)
					}
				}
				args = filtered
			}
			validate := append([]string(nil), args...)
			validate[0] = "validate"
			result(t, invoke(t, base, append(args, "--check")...), 3)
			result(t, invoke(t, base, validate...), 0)
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("fresh check/validate wrote", err)
			}
			result(t, invoke(t, base, args...), 0)
			original := read(t, input)
			r, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: checked, ModelStyle: style, HoistSharedShapes: true, DeduplicateModels: true, MergeIdenticalModels: true, LockHoistedNames: true}})
			if err != nil {
				t.Fatal(err)
			}
			aliases, actualDefault := 0, false
			for _, file := range r.Artifacts.Files {
				if string(read(t, filepath.Join(output, file.Path))) != file.Content {
					t.Fatal("optimized actual core/native source mismatch", file.Path)
				}
				if strings.Contains(file.Path, "/api/models/") && strings.Contains(file.Content, "public typealias ") {
					aliases++
				}
				if strings.Contains(file.Path, "/models/") && !strings.Contains(file.Path, "/api/") && strings.Contains(file.Content, "public val value:") && strings.Contains(file.Content, "kotlin.Float?") {
					actualDefault = true
				}
			}
			if aliases < 2 || !actualDefault {
				t.Fatal("optimizer did not produce real shared scalar tree and exported aliases", aliases, actualDefault)
			}
			if !bytes.Equal(r.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
				t.Fatal("actual core/native lock mismatch")
			}
			for _, name := range []string{"alpha", "beta"} {
				if !bytes.Contains(r.Artifacts.NameLock, []byte("/paths/~1"+name+"/post/requestBody/content/application~1json/schema")) {
					t.Fatal("lost original occurrence", name)
				}
			}
			put(t, filepath.Join(output, "Notes.kt"), []byte("handwritten"))
			before := snapshot(t, output)
			put(t, input, bytes.ReplaceAll(original, []byte(`{"z":{"z":null,"a":0.5},"a":{"/~":1}}`), []byte(`{"a":{"/~":1},"z":{"a":0.5,"z":null}}`)))
			result(t, invoke(t, base, args...), 0)
			if err := os.Chmod(output, 0500); err != nil {
				t.Fatal(err)
			}
			result(t, invoke(t, base, append(args, "--check")...), 0)
			result(t, invoke(t, base, validate...), 0)
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("canonical no-op/readonly check/validate changed bytes/modes/mtimes")
			}
			if err := os.Chmod(output, 0700); err != nil {
				t.Fatal(err)
			}
			cases := []struct{ old, replacement, code, tail string }{
				{`"a":0.5`, `"a":1`, "E_NAME_LOCK_CHANGED", ""},
				{`"a":0.5`, `"a":0.500000001`, "E_SCHEMA_INVALID", "/default/z/a"},
				{`"a":0.5`, `"a":true`, "E_SCHEMA_INVALID", "/default/z/a"},
				{`"/~":1`, `"/~":2`, "E_SCHEMA_INVALID", "/default/a/~1~0"},
				{`"enum":[0.5,1]`, `"enum":[0.5,0.5]`, "E_SCHEMA_INVALID", "/additionalProperties/additionalProperties/enum/1"},
				{`"nullable":true`, `"nullable":false`, "E_SCHEMA_INVALID", "/default/z/z"},
			}
			if checked {
				cases = append(cases,
					struct{ old, replacement, code, tail string }{`"enum":[0.5,1]`, `"enum":[0.5,1,2]`, "E_NAME_LOCK_CHANGED", ""},
					struct{ old, replacement, code, tail string }{`"format":"float"`, `"format":"float","minimum":0`, "E_NAME_LOCK_CHANGED", ""},
					struct{ old, replacement, code, tail string }{`{"additionalProperties":{"additionalProperties"`, `{"maxProperties":3,"additionalProperties":{"additionalProperties"`, "E_NAME_LOCK_CHANGED", ""},
				)
			}
			for _, tc := range cases {
				changed := bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement))
				if bytes.Equal(changed, original) {
					t.Fatal("missing replacement", tc.old)
				}
				put(t, input, changed)
				for _, command := range [][]string{args, append(append([]string(nil), args...), "--check"), validate} {
					failure := fails(t, invoke(t, base, command...), 1, tc.code)
					if tc.tail != "" && (failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/paths/~1alpha/post/requestBody/content/application~1json/schema/properties/value"+tc.tail) || !reflect.DeepEqual(before, snapshot(t, output)) {
						t.Fatal("original error/preservation", tc, failure)
					}
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsNativeOriginalErrorsAndSuppression(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		put(t, p.Input, nestedNativeEnumDocument(nestedNativeEnumField("LL", `{"type":"integer","enum":[1,2]}`, `[[1]]`, "")))
		args := nestedNativeEnumArgs(p.Input, p.Output, checked, core.DataClass)
		result(t, invoke(t, p.Root, args...), 0)
		put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
		before := snapshot(t, p.Output)
		prefix := "/components/schemas/Holder/properties/value"
		for _, tc := range []struct{ leaf, value, code, tail string }{
			{`{"type":"integer","enum":[1]}`, `[{"a/~":2}]`, "E_SCHEMA_INVALID", "/default/0/a~1~0"},
			{`{"type":"integer","enum":[1]}`, `[{"z":2,"a":3}]`, "E_SCHEMA_INVALID", "/default/0/z"},
			{`{"type":"integer","enum":[1]}`, `[{"a":true}]`, "E_SCHEMA_INVALID", "/default/0/a"},
			{`{"type":"number","format":"float","enum":[1]}`, `[{"a":1.000000001}]`, "E_SCHEMA_INVALID", "/default/0/a"},
			{`{"type":"number","enum":[1]}`, `[{"a":1.00000000000000001}]`, "E_SCHEMA_INVALID", "/default/0/a"},
			{`{"type":"boolean","enum":[true]}`, `[{"a":null}]`, "E_SCHEMA_INVALID", "/default/0/a"},
			{`{"type":"integer","enum":[]}`, `[]`, "E_SCHEMA_INVALID", "/items/additionalProperties/enum"},
			{`{"type":"boolean","enum":[true,"bad"]}`, `[]`, "E_SCHEMA_INVALID", "/items/additionalProperties/enum/1"},
			{`{"type":"integer","enum":[1],"default":1}`, `[]`, "E_SCHEMA_UNSUPPORTED", "/items/additionalProperties/default"},
		} {
			doc := nestedNativeEnumDocument(nestedNativeEnumField("LM", tc.leaf, tc.value, ""))
			r, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: doc, Options: core.GenerationOptions{PackageName: "org.example.catalog", Validation: checked}})
			if err == nil || !reflect.DeepEqual(r, core.GenerationReport{}) {
				t.Fatal("core returned partial success", tc, err)
			}
			diagnostics, ok := err.(core.Diagnostics)
			if !ok || len(diagnostics) != 1 || diagnostics[0].Code != tc.code || diagnostics[0].Pointer == nil || *diagnostics[0].Pointer != prefix+tc.tail {
				t.Fatal("core original diagnostic", tc, err)
			}
			put(t, p.Input, doc)
			failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != prefix+tc.tail || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("native original diagnostic/preservation", tc, failure)
			}
		}
		field := nestedNativeEnumField("LM", `{"type":"integer","enum":[1]}`, `[{"a/~":2}]`, "")
		for _, tc := range []struct{ document, pointer string }{
			{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":` + field + `},"required":["value"]}}}}`, prefix + "/default/0/a~1~0"},
			{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}}}`, "/components/schemas/Holder/allOf/0/properties/value/default/0/a~1~0"},
			{`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(field, `"type":"array"`, `"type":"array","nullable":true`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`, "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/value/default/0/a~1~0"},
		} {
			put(t, p.Input, []byte(tc.document))
			code, pointer := "E_SCHEMA_INVALID", tc.pointer
			if checked && strings.Contains(pointer, "/allOf/") {
				// Existing NoDrop rejects the original enforcing vocabulary before
				// the discarded annotation can acquire a scratch default plan.
				code = "E_SCHEMA_UNSUPPORTED"
				pointer = "/components/schemas/Holder/allOf/0/properties/value/items/additionalProperties/enum"
			}
			failure := fails(t, invoke(t, p.Root, args...), 1, code)
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("suppression lost prevalidation/preservation", failure)
			}
		}
	}
}
