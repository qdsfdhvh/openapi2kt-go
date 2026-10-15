package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedBuiltinLockInput(t testing.TB, base string) string {
	t.Helper()
	path := filepath.Join(base, "input.json")
	field := `{"additionalProperties":{"type":"array","items":{"type":"number","format":"float"}},"default":{"b":[1.25],"a":[1]}}`
	op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{"description":"OK"}}}}`
	put(t, path, []byte(`{"openapi":"3.1.0","paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return path
}

func TestNestedBuiltinCollectionDefaultsNativeCoreLockNoopAndFailurePreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := nestedBuiltinLockInput(t, base)
		output := filepath.Join(base, "out")
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		result(t, invoke(t, base, append(args, "--check")...), 3)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		result(t, invoke(t, base, args...), 0)
		// Compare fresh actual native source and state with the pure core output.
		report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: read(t, input), Format: core.Auto, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true, ModelStyle: style}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range report.Artifacts.Files {
			if string(read(t, filepath.Join(output, file.Path))) != file.Content {
				t.Fatal("native/core differ", file.Path)
			}
		}
		if !bytes.Equal(report.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
			t.Fatal("native/core state differs")
		}
		before := snapshot(t, output)
		original := read(t, input)
		reordered := bytes.ReplaceAll(original, []byte(`{"b":[1.25],"a":[1]}`), []byte(`{"a":[1.000000001],"b":[1.250]}`))
		put(t, input, reordered)
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("canonical noop/check/validate changed state/mtime")
		}
		for _, tc := range []struct{ old, replacement, code string }{
			{`"b":[1.25]`, `"b":[2]`, "E_NAME_LOCK_CHANGED"},
			{`"b":[1.25]`, `"b":[false]`, "E_SCHEMA_INVALID"},
			{`"type":"number","format":"float"`, `"type":"number","format":"float","minimum":0`, "E_NAME_LOCK_CHANGED"},
			{`"type":"array","items"`, `"type":"array","prefixItems":[],"items"`, "E_SCHEMA_UNSUPPORTED"},
		} {
			put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
			fails(t, invoke(t, base, args...), 1, tc.code)
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("failed annotation changed state", tc)
			}
		}
	}
}
