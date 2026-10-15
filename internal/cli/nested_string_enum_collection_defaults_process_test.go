package cli_test

import (
	"bytes"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func nestedStringEnumLockInput(t testing.TB, base string) string {
	t.Helper()
	path := filepath.Join(base, "input.json")
	field := `{"additionalProperties":{"additionalProperties":{"enum":["one","two"],"nullable":true}},"default":{"z":{"z":null,"a":"two"},"a":{"/~":"one"}}}`
	op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	put(t, path, []byte(`{"openapi":"3.1.0","paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return path
}

func TestNestedStringEnumCollectionDefaultsNativeCoreParityReadonlyAndPreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			base := t.TempDir()
			input := nestedStringEnumLockInput(t, base)
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
			result(t, invoke(t, base, append(args, "--check")...), 3)
			validate := append([]string(nil), args...)
			validate[0] = "validate"
			result(t, invoke(t, base, validate...), 0)
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("fresh check/validate wrote", err)
			}
			result(t, invoke(t, base, args...), 0)
			original := read(t, input)
			report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: checked, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true, ModelStyle: style}})
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Artifacts.Files {
				if string(read(t, filepath.Join(output, f.Path))) != f.Content {
					t.Fatal("source parity", f.Path)
				}
			}
			if !bytes.Equal(report.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
				t.Fatal("lock parity")
			}
			for _, pointer := range []string{"/paths/~1alpha/get/responses/200/content/application~1json/schema", "/paths/~1beta/get/responses/200/content/application~1json/schema"} {
				if !bytes.Contains(report.Artifacts.NameLock, []byte(pointer)) {
					t.Fatal("lost occurrence", pointer)
				}
			}
			put(t, filepath.Join(output, "Notes.kt"), []byte("handwritten"))
			before := snapshot(t, output)
			put(t, input, bytes.ReplaceAll(original, []byte(`{"z":{"z":null,"a":"two"},"a":{"/~":"one"}}`), []byte(`{"a":{"/~":"one"},"z":{"a":"two","z":null}}`)))
			result(t, invoke(t, base, args...), 0)
			if err := os.Chmod(output, 0500); err != nil {
				t.Fatal(err)
			}
			result(t, invoke(t, base, append(args, "--check")...), 0)
			result(t, invoke(t, base, validate...), 0)
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("canonical noop/readonly attrs drift")
			}
			if err := os.Chmod(output, 0700); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ old, replacement, code, tail string }{
				{`"a":"two"`, `"a":"one"`, "E_NAME_LOCK_CHANGED", ""},
				{`"a":"two"`, `"a":"future"`, "E_SCHEMA_INVALID", "/default/z/a"},
				{`"a":"two"`, `"a":1`, "E_SCHEMA_INVALID", "/default/z/a"},
				{`"one","two"`, `"two","one"`, "E_NAME_LOCK_CHANGED", ""},
				{`"enum":["one","two"]`, `"enum":["one","one"]`, "E_SCHEMA_INVALID", "/additionalProperties/additionalProperties/enum/1"},
				{`"enum":["one","two"]`, `"enum":["one","two"],"minLength":1`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/additionalProperties/minLength"},
			} {
				put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
				for _, command := range [][]string{args, append(append([]string(nil), args...), "--check"), validate} {
					failure := fails(t, invoke(t, base, command...), 1, tc.code)
					if tc.tail != "" && (failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/paths/~1alpha/get/responses/200/content/application~1json/schema/properties/value"+tc.tail) || !reflect.DeepEqual(before, snapshot(t, output)) {
						t.Fatal("failure/pointer/preservation", tc, failure)
					}
				}
			}
		}
	}
}

func TestNestedStringEnumCollectionDefaultsNativeRequiredPatchDiscardedAndCaps(t *testing.T) {
	p := newProject(t)
	good := `{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"enum":["one"]}},"default":[["one"]]}}}}}}`
	put(t, p.Input, []byte(good))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
	before := snapshot(t, p.Output)
	field := `{"type":"array","items":{"additionalProperties":{"enum":["one"]}},"default":[{"a/~":"future"}]}`
	for _, tc := range []struct{ input, pointer string }{
		{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":` + field + `},"required":["value"]}}}}`, "/components/schemas/Holder/properties/value/default/0/a~1~0"},
		{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}}}`, "/components/schemas/Holder/allOf/0/properties/value/default/0/a~1~0"},
		{`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(field, `"type":"array"`, `"type":"array","nullable":true`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`, "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/value/default/0/a~1~0"},
	} {
		put(t, p.Input, []byte(tc.input))
		failure := fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
		if *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal(failure)
		}
	}
	token := strings.Repeat("a", 32763)
	put(t, p.Input, []byte(strings.ReplaceAll(good, `"one"`, `"`+token+`"`)))
	fails(t, p.generate(t), 1, "E_MODEL_LIMIT")
	put(t, p.Input, bytes.Repeat([]byte(" "), core.MaxDocumentBytes+1))
	fails(t, p.generate(t), 1, "E_INPUT_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("resource failure changed old output")
	}
}
