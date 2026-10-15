package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedStringEnumLockInput(t testing.TB, base string) string {
	t.Helper()
	field := `{"$ref":"#/components/schemas/Rows","default":{"z":[],"a":["on",null]}}`
	op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{}}}}`
	input := filepath.Join(base, "input.json")
	put(t, input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":"string","enum":["on","off"]},"Rows":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Status","nullable":true}}}}},"paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return input
}

func TestReferencedStringEnumCollectionFieldDefaultsNativeLockAndPreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := referencedStringEnumLockInput(t, base)
		output := filepath.Join(base, "out")
		original := read(t, input)
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		validate := append([]string{}, args...)
		validate[0] = "validate"
		result(t, invoke(t, base, append(args, "--check")...), 3)
		result(t, invoke(t, base, validate...), 0)
		result(t, invoke(t, base, args...), 0)
		got, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range got.Artifacts.Files {
			if string(read(t, filepath.Join(output, f.Path))) != f.Content {
				t.Fatal("core/native source parity", f.Path)
			}
		}
		lock := filepath.Join(output, ".openapi-kt-names.json")
		if !bytes.Equal(got.Artifacts.NameLock, read(t, lock)) {
			t.Fatal("raw lock parity")
		}
		put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, output)
		put(t, input, bytes.ReplaceAll(original, []byte(`{"z":[],"a":["on",null]}`), []byte(`{"a":["on",null],"z":[]}`)))
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("canonical/no-op/readonly attributes changed")
		}
		for _, tc := range []struct{ old, replacement, code string }{
			{`"a":["on",null]`, `"a":["off",null]`, "E_NAME_LOCK_CHANGED"},
			{`"a":["on",null]`, `"a":["gone",null]`, "E_SCHEMA_INVALID"},
			{`"a":["on",null]`, `"a":["on"]`, "E_NAME_LOCK_CHANGED"},
			{`"type":"array","items"`, `"type":"array","maxItems":2,"items"`, "E_NAME_LOCK_CHANGED"},
			{`"type":"array","items"`, `"type":"array","default":[],"items"`, "E_SCHEMA_UNSUPPORTED"},
			{`"$ref":"#/components/schemas/Status","nullable":true`, `"$ref":"#/components/schemas/Status","nullable":true,"minLength":1`, "E_SCHEMA_UNSUPPORTED"},
		} {
			put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
				fails(t, invoke(t, base, command...), 1, tc.code)
			}
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("failed candidate changed old output", tc)
			}
		}
		put(t, input, original)
		put(t, lock, append(read(t, lock), ' '))
		modified := snapshot(t, output)
		fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(modified, snapshot(t, output)) {
			t.Fatal("modified raw lock overwritten")
		}
	}
}

func TestReferencedStringEnumCollectionFieldDefaultsNativePointerAndInputCaps(t *testing.T) {
	p := newProject(t)
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":"string","enum":["on","off"]},"Rows":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Status","nullable":true}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[{"a/~":"on"}]}}}}}}`)
	put(t, p.Input, original)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, bytes.Replace(original, []byte(`"a/~":"on"`), []byte(`"a/~":"gone"`), 1))
	failed := fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
	if *failed.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/v/default/0/a~1~0" {
		t.Fatal(failed)
	}
	var huge []byte
	for len(huge) < 32762 {
		huge = append(huge, 'a')
	}
	put(t, p.Input, bytes.Replace(original, []byte(`"a/~":"on"`), append(append([]byte(`"a/~":"`), huge...), '"'), 1))
	fails(t, p.generate(t), 1, "E_MODEL_LIMIT")
	put(t, p.Input, bytes.Replace(original, []byte(`"on"`), append(append([]byte{'"'}, bytes.Repeat([]byte("a"), 8*1024*1024)...), '"'), 1))
	fails(t, p.generate(t), 1, "E_INPUT_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed value/budget changed previous output")
	}
}
