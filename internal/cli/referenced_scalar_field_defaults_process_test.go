package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedScalarFieldDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Count":{"type":"integer","format":"int64"},"Holder":{"properties":{"count":{"$ref":"#/components/schemas/Count","default":2}}}}}}`)
	put(t, p.Input, bytes.Replace(original, []byte(`,"default":2`), nil, 1))
	result(t, p.generate(t), 0)
	put(t, p.Input, original)
	result(t, p.generate(t), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/catalog/models/Holder.kt"))), "kotlin.Long? = 2L") {
		t.Fatal("actual scalar initializer missing")
	}
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op/check/validate mutated output")
	}
	for _, tc := range []struct{ old, replacement, code, pointer string }{
		{`"default":2`, `"default":9223372036854775808`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/count/default"},
		{`"format":"int64"`, `"format":"int64","default":1`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Count/default"},
		{`"#/components/schemas/Count"`, `"#/components/schemas/Missing"`, "E_REF_MISSING", "/components/schemas/Holder/properties/count"},
		{`"default":2`, `"default":null`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/count/default"},
	} {
		put(t, p.Input, bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1))
		failed := fails(t, p.generate(t), 1, tc.code)
		if *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal(tc, failed)
		}
	}
}

func TestReferencedScalarFieldDefaultsNativeCoreLock(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := filepath.Join(base, "input.json")
		output := filepath.Join(base, "out")
		field := `{"$ref":"#/components/schemas/Value","default":1}`
		op := `{"post":{"operationId":"alpha","requestBody":{"content":{"application/json":{"schema":{"properties":{"v":` + field + `}}}}},"responses":{"204":{}}}}`
		original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"number","format":"float"}}},"paths":{"/alpha":` + op + `,"/beta":` + strings.ReplaceAll(op, `"alpha"`, `"beta"`) + `}}`)
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
				t.Fatal("core/native mismatch", file.Path)
			}
		}
		lock := filepath.Join(output, ".openapi-kt-names.json")
		if !bytes.Equal(got.Artifacts.NameLock, read(t, lock)) {
			t.Fatal("lock parity")
		}
		put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, output)
		put(t, input, bytes.ReplaceAll(original, []byte(`"default":1`), []byte(`"default":1.000000001`)))
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("mapped no-op mutated bytes/attributes")
		}
		put(t, input, bytes.ReplaceAll(original, []byte(`"default":1`), []byte(`"default":2`)))
		fails(t, invoke(t, base, args...), 1, "E_NAME_LOCK_CHANGED")
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("lock drift changed old output")
		}
		put(t, input, original)
		raw := read(t, lock)
		put(t, lock, append(raw, ' '))
		modified := snapshot(t, output)
		fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(modified, snapshot(t, output)) {
			t.Fatal("modified lock overwritten")
		}
	}
}
