package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedBytesDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bytes":{"type":"string","format":"binary"},"Holder":{"properties":{"a/~":{"$ref":"#/components/schemas/Bytes","default":[-128,127]}}}}}}`)
	put(t, p.Input, bytes.Replace(original, []byte(`,"default":[-128,127]`), nil, 1))
	result(t, p.generate(t), 0)
	put(t, p.Input, original)
	result(t, p.generate(t), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/catalog/models/Holder.kt"))), "`openapi-kt default tree Scalar ByteArray`(\"[-128,127]\")") {
		t.Fatal("missing byte initializer")
	}
	put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--check"), 0)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op changed attributes")
	}
	for _, tc := range []struct{ old, value, code, pointer string }{
		{`"default":[-128,127]`, `"default":[-128,128]`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/default/1"},
		{`"default":[-128,127]`, `"default":"AQI="`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/default"},
		{`"default":[-128,127]`, `"default":[null]`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/default/0"},
		{`"format":"binary"`, `"format":"binary","default":[1]`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Bytes/default"},
		{`"#/components/schemas/Bytes"`, `"#/components/schemas/Missing"`, "E_REF_MISSING", "/components/schemas/Holder/properties/a~1~0"},
	} {
		put(t, p.Input, bytes.Replace(original, []byte(tc.old), []byte(tc.value), 1))
		for _, extra := range [][]string{nil, {"--check"}} {
			failed := fails(t, p.generate(t, extra...), 1, tc.code)
			if *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal(tc, failed)
			}
		}
	}
}

func TestReferencedBytesDefaultsNativeCoreAndLockedIdentity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input, output := filepath.Join(base, "input.json"), filepath.Join(base, "out")
		field := `{"$ref":"#/components/schemas/Bytes","default":[1]}`
		op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"v":` + field + `}}}}}}}}`
		original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bytes":{"type":"string","format":"byte","nullable":true}}},"paths":{"/alpha":` + op + `,"/beta":` + strings.ReplaceAll(op, `"alpha"`, `"beta"`) + `}}`)
		put(t, input, original)
		args := append(lockArgs(input, output), "--model-style="+string(style), "--merge-identical-models")
		result(t, invoke(t, base, append(args, "--check")...), 3)
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
		if !bytes.Equal(got.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
			t.Fatal("lock parity")
		}
		before := snapshot(t, output)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		put(t, input, bytes.ReplaceAll(original, []byte(`"default":[1]`), []byte(`"default":[2]`)))
		fails(t, invoke(t, base, args...), 1, "E_NAME_LOCK_CHANGED")
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("changed default overwrote locked output")
		}
	}
}
