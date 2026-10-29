package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestReferencedNativeEnumScalarFieldDefaultsNativeLockAndPreservation(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "input.json")
	output := filepath.Join(base, "out")
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","format":"int32","enum":[1,2]}}},"paths":{"/a":{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"v":{"$ref":"#/components/schemas/Code","default":1}}}}}}}}},"/b":{"get":{"operationId":"beta","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"v":{"$ref":"#/components/schemas/Code","default":1}}}}}}}}}}}`)
	put(t, input, original)
	args := lockArgs(input, output)
	validate := append([]string{}, args...)
	validate[0] = "validate"
	result(t, invoke(t, base, append(args, "--check")...), 3)
	result(t, invoke(t, base, args...), 0)
	result(t, invoke(t, base, append(args, "--check")...), 0)
	result(t, invoke(t, base, validate...), 0)
	report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range report.Artifacts.Files {
		if string(read(t, filepath.Join(output, file.Path))) != file.Content {
			t.Fatal("core/native parity", file.Path)
		}
	}
	if !bytes.Equal(report.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
		t.Fatal("raw lock parity")
	}
	put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
	before := snapshot(t, output)
	for _, tc := range []struct{ old, replacement, code string }{
		{`"default":1`, `"default":2`, "E_NAME_LOCK_CHANGED"},
		{`"default":1`, `"default":9`, "E_SCHEMA_INVALID"},
		{`"default":1`, `"default":null`, "E_SCHEMA_INVALID"},
		{`"enum":[1,2]`, `"enum":[1,2],"default":9`, "E_SCHEMA_INVALID"},
	} {
		put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
		for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
			fails(t, invoke(t, base, command...), 1, tc.code)
		}
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("failed scalar default changed output", tc)
		}
	}
	put(t, input, original)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("no-op changed output")
	}
	// Preserve the exact former refusal as an independent root-factory positive.
	put(t, input, bytes.ReplaceAll(original, []byte(`"enum":[1,2]`), []byte(`"enum":[1,2],"default":1`)))
	factoryOutput := filepath.Join(base, "factory-out")
	factoryArgs := lockArgs(input, factoryOutput)
	factoryValidate := append([]string{}, factoryArgs...)
	factoryValidate[0] = "validate"
	for _, command := range [][]string{factoryValidate, factoryArgs, append(append([]string{}, factoryArgs...), "--check")} {
		result(t, invoke(t, base, command...), 0)
	}
	if !bytes.Contains(read(t, filepath.Join(factoryOutput, "org/example/dedup/models/CodeDto.kt")), []byte("CodeDto default")) {
		t.Fatal("independent native component factory missing")
	}
	put(t, input, original)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("independent root factory changed original consumers or lock")
	}
}
