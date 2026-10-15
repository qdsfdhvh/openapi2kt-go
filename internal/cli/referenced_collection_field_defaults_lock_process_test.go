package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedDefaultLockInput(t testing.TB, base string) string {
	t.Helper()
	field := `{"$ref":"#/components/schemas/Values","default":{"b":[1.25],"a":[1]}}`
	op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{}}}}`
	input := filepath.Join(base, "input.json")
	put(t, input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Values":{"additionalProperties":{"type":"array","items":{"type":"number","format":"float"}}}}},"paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return input
}

func TestReferencedCollectionFieldDefaultsNativeCoreLockIdentity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := referencedDefaultLockInput(t, base)
		output := filepath.Join(base, "out")
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		result(t, invoke(t, base, append(args, "--check")...), 3)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		result(t, invoke(t, base, args...), 0)
		got, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: read(t, input), Format: core.Auto, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range got.Artifacts.Files {
			if string(read(t, filepath.Join(output, file.Path))) != file.Content {
				t.Fatal("native/core source mismatch", file.Path)
			}
		}
		if !bytes.Equal(got.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
			t.Fatal("native/core lock mismatch")
		}
		before := snapshot(t, output)
		original := read(t, input)
		put(t, input, bytes.ReplaceAll(original, []byte(`{"b":[1.25],"a":[1]}`), []byte(`{"a":[1.000000001],"b":[1.250]}`)))
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("canonical no-op/check/validate mutated output")
		}
		for _, tc := range []struct{ old, replacement, code string }{
			{`"b":[1.25]`, `"b":[2]`, "E_NAME_LOCK_CHANGED"},
			{`"b":[1.25]`, `"b":[false]`, "E_SCHEMA_INVALID"},
			{`"type":"number","format":"float"`, `"type":"number","format":"float","minimum":0`, "E_NAME_LOCK_CHANGED"},
			{`"type":"array","items"`, `"type":"array","default":[],"items"`, "E_SCHEMA_UNSUPPORTED"},
		} {
			put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
			fails(t, invoke(t, base, args...), 1, tc.code)
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("failed default replaced output", tc)
			}
		}
	}
}

func TestReferencedCollectionFieldDefaultsNativeOriginalMapAndLimits(t *testing.T) {
	p := newProject(t)
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bag","default":{}}}},"Bag":{"type":"object","additionalProperties":{"type":"integer"}}}}}`)
	put(t, p.Input, original)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	for _, size := range []int{32769, 8 * 1024 * 1024} {
		var input []byte
		code := "E_MODEL_LIMIT"
		if size == 32769 {
			input = bytes.Replace(original, []byte(`"default":{}`), []byte(`"default":{"a":"`+strings.Repeat("a", size)+`"}`), 1)
		} else {
			input = bytes.Replace(original, []byte(`"openapi":`), []byte(`"description":"`+strings.Repeat("a", size)+`","openapi":`), 1)
			code = "E_INPUT_LIMIT"
		}
		put(t, p.Input, input)
		fails(t, p.generate(t), 1, code)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("budget failure replaced output", size)
		}
	}
}
