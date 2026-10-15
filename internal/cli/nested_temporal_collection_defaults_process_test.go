package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedTemporalLockInput(t testing.TB, base string) string {
	t.Helper()
	path := filepath.Join(base, "input.json")
	field := `{"additionalProperties":{"type":"array","items":{"type":"string","format":"date-time"}},"default":{"b":["2024-02-29T12:34:56+02:30"],"a":["2024-02-29T00:00:00Z"]}}`
	op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{"description":"OK"}}}}`
	put(t, path, []byte(`{"openapi":"3.1.0","paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return path
}

func TestNestedTemporalCollectionDefaultsNativeCoreLockReadonlyAndFailurePreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := nestedTemporalLockInput(t, base)
		output := filepath.Join(base, "out")
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		result(t, invoke(t, base, append(args, "--check")...), 3)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		result(t, invoke(t, base, args...), 0)
		original := read(t, input)
		report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Format: core.Auto, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true, ModelStyle: style}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range report.Artifacts.Files {
			if string(read(t, filepath.Join(output, file.Path))) != file.Content {
				t.Fatal("native/core source differs", file.Path)
			}
		}
		if !bytes.Equal(report.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
			t.Fatal("native/core state differs")
		}
		if !bytes.Contains(report.Artifacts.NameLock, []byte("/paths/~1alpha/post/requestBody/content/application~1json/schema")) || !bytes.Contains(report.Artifacts.NameLock, []byte("/paths/~1beta/post/requestBody/content/application~1json/schema")) {
			t.Fatal("original lock occurrences lost")
		}
		put(t, filepath.Join(output, "Notes.kt"), []byte("handwritten"))
		before := snapshot(t, output)
		put(t, input, bytes.ReplaceAll(original, []byte(`{"b":["2024-02-29T12:34:56+02:30"],"a":["2024-02-29T00:00:00Z"]}`), []byte(`{"a":["2024-02-29T00:00:00Z"],"b":["2024-02-29T12:34:56+02:30"]}`)))
		result(t, invoke(t, base, args...), 0)
		if err := os.Chmod(output, 0500); err != nil {
			t.Fatal(err)
		}
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("canonical no-op/readonly check/validate changed state/attributes")
		}
		if err := os.Chmod(output, 0700); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ old, replacement, code, pointer string }{
			{"2024-02-29T12:34:56+02:30", "2024-02-29T10:04:56Z", "E_NAME_LOCK_CHANGED", ""},
			{"2024-02-29T12:34:56+02:30", "2023-02-29T12:34:56+02:30", "E_SCHEMA_INVALID", "/paths/~1alpha/post/requestBody/content/application~1json/schema/properties/value/default/b/0"},
			{"2024-02-29T12:34:56+02:30", "2024-02-29T12:34:56.1234567891Z", "E_SCHEMA_INVALID", "/paths/~1alpha/post/requestBody/content/application~1json/schema/properties/value/default/b/0"},
			{"2024-02-29T12:34:56+02:30", "2024-02-29T12:34:56-18:01", "E_SCHEMA_INVALID", "/paths/~1alpha/post/requestBody/content/application~1json/schema/properties/value/default/b/0"},
			{`"type":"array","items"`, `"type":"array","maxItems":2,"items"`, "E_NAME_LOCK_CHANGED", ""},
			{`"type":"string","format":"date-time"`, `"type":"string","format":"date-time","minLength":1`, "E_SCHEMA_UNSUPPORTED", "/paths/~1alpha/post/requestBody/content/application~1json/schema/properties/value/additionalProperties/items/minLength"},
		} {
			put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
			for _, command := range [][]string{args, append(append([]string(nil), args...), "--check"), validate} {
				failure := fails(t, invoke(t, base, command...), 1, tc.code)
				pointerOK := tc.pointer == "" && failure.Diagnostics[0].Pointer == nil || tc.pointer != "" && failure.Diagnostics[0].Pointer != nil && *failure.Diagnostics[0].Pointer == tc.pointer
				if !pointerOK || !reflect.DeepEqual(before, snapshot(t, output)) {
					t.Fatal("failure pointer/preservation", tc, failure)
				}
			}
		}
	}
}

func TestNestedTemporalCollectionDefaultsNativeRequiredPatchDiscardedAndLimits(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/nested-temporal-collection-defaults.json")))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
	before := snapshot(t, p.Output)
	field := `{"type":"array","items":{"additionalProperties":{"type":"string","format":"date"}},"default":[{"a/~":"2023-02-29"}]}`
	for _, tc := range []struct{ input, code, pointer string }{
		{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":` + field + `},"required":["value"]}}}}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/0/a~1~0"},
		{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}}}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/value/default/0/a~1~0"},
		{`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(field, `"type":"array"`, `"type":"array","nullable":true`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`, "E_SCHEMA_INVALID", "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/value/default/0/a~1~0"},
		{`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"array","items":{"additionalProperties":{"type":"string","format":"date"}},"default":[{"a/~":"` + strings.Repeat("a", 32769) + `"}]}}}}}}`, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/value/default/0/a~1~0"},
	} {
		put(t, p.Input, []byte(tc.input))
		failure := fails(t, p.generate(t), 1, tc.code)
		if *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal(tc.code, failure)
		}
	}
	put(t, p.Input, bytes.Repeat([]byte(" "), core.MaxDocumentBytes+1))
	fails(t, p.generate(t), 1, "E_INPUT_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("8MiB error changed old output")
	}
}
