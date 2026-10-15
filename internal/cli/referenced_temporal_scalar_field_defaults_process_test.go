package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedTemporalScalarFieldDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"a/~":{"$ref":"#/components/schemas/Day","default":"2024-02-29"}}}}}}`)
	put(t, p.Input, bytes.Replace(original, []byte(`,"default":"2024-02-29"`), nil, 1))
	result(t, p.generate(t), 0)
	put(t, p.Input, original)
	result(t, p.generate(t), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/catalog/models/Holder.kt"))), "`openapi-kt default date`(\"2024-02-29\")") {
		t.Fatal("typed initializer missing")
	}
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op/readonly changed bytes attributes")
	}
	for _, tc := range []struct{ old, replacement, code, pointer string }{
		{`"default":"2024-02-29"`, `"default":"2023-02-29"`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/default"},
		{`"default":"2024-02-29"`, `"default":null`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/default"},
		{`"format":"date"`, `"format":"date","default":"2024-02-29"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Day/default"},
		{`"format":"date"`, `"format":"date","minLength":0`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Day/minLength"},
		{`"#/components/schemas/Day"`, `"#/components/schemas/Missing"`, "E_REF_MISSING", "/components/schemas/Holder/properties/a~1~0"},
	} {
		put(t, p.Input, bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1))
		for _, extra := range [][]string{nil, {"--check"}} {
			failed := fails(t, p.generate(t, extra...), 1, tc.code)
			if *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal(tc, failed)
			}
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsNativeCoreLockLexeme(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input, output := filepath.Join(base, "input.json"), filepath.Join(base, "out")
		field := `{"$ref":"#/components/schemas/Stamp","default":"2024-02-29T12:34:56+02:30"}`
		op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"v":` + field + `}}}}}}}}`
		original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Stamp":{"type":"string","format":"date-time","nullable":true}}},"paths":{"/alpha":` + op + `,"/beta":` + strings.ReplaceAll(op, `"alpha"`, `"beta"`) + `}}`)
		put(t, input, original)
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		result(t, invoke(t, base, append(args, "--check")...), 3)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		result(t, invoke(t, base, args...), 0)
		got, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true, MergeIdenticalModels: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range got.Artifacts.Files {
			if string(read(t, filepath.Join(output, file.Path))) != file.Content {
				t.Fatal("core/native parity", file.Path)
			}
		}
		lock := filepath.Join(output, ".openapi-kt-names.json")
		if !bytes.Equal(got.Artifacts.NameLock, read(t, lock)) {
			t.Fatal("lock parity")
		}
		put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, output)
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("lock noop attributes")
		}
		put(t, input, bytes.ReplaceAll(original, []byte("2024-02-29T12:34:56+02:30"), []byte("2024-02-29T10:04:56Z")))
		for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
			fails(t, invoke(t, base, command...), 1, "E_NAME_LOCK_CHANGED")
		}
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("equivalent time lexeme overwrote old output")
		}
		put(t, input, original)
		put(t, lock, append(read(t, lock), ' '))
		modified := snapshot(t, output)
		fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(modified, snapshot(t, output)) {
			t.Fatal("modified lock overwritten")
		}
	}
}
