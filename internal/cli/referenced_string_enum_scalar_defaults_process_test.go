package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestReferencedStringEnumScalarDefaultsNativeLockAndPreservation(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "input.json")
	output := filepath.Join(base, "out")
	original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":"string","enum":["on","off"]}}},"paths":{"/a":{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"a/~":{"$ref":"#/components/schemas/Status","default":"on"}}}}}}}}},"/b":{"get":{"operationId":"beta","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"a/~":{"$ref":"#/components/schemas/Status","default":"on"}}}}}}}}}}}`)
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
	put(t, input, bytes.ReplaceAll(original, []byte(`"default":"on"`), []byte(`"default":"\u006fn"`)))
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("equivalent String spelling changed identity")
	}
	for _, tc := range []struct{ old, replacement, code string }{
		{`"default":"on"`, `"default":"off"`, "E_NAME_LOCK_CHANGED"},
		{`"default":"on"`, `"default":"future"`, "E_SCHEMA_INVALID"},
		{`"default":"on"`, `"default":null`, "E_SCHEMA_INVALID"},
		{`"enum":["on","off"]`, `"enum":["on","off"],"default":"future"`, "E_SCHEMA_INVALID"},
		{`"enum":["on","off"]`, `"enum":["on","off"],"minLength":1`, "E_NAME_LOCK_CHANGED"},
	} {
		put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
		for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
			result := fails(t, invoke(t, base, command...), 1, tc.code)
			if tc.replacement == `"default":"future"` && (result.Diagnostics[0].Pointer == nil || *result.Diagnostics[0].Pointer != "/paths/~1a/get/responses/200/content/application~1json/schema/properties/a~1~0/default") {
				t.Fatal("original escaped default pointer", result)
			}
		}
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("failed candidate changed output", tc)
		}
	}
	put(t, input, original)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("no-op changed output")
	}
	// Keep the original accepted root annotation independent of the field default.
	put(t, input, bytes.ReplaceAll(original, []byte(`"enum":["on","off"]`), []byte(`"enum":["on","off"],"default":"on"`)))
	factoryOutput := filepath.Join(base, "factory-out")
	factoryArgs := lockArgs(input, factoryOutput)
	factoryValidate := append([]string{}, factoryArgs...)
	factoryValidate[0] = "validate"
	for _, command := range [][]string{factoryValidate, factoryArgs, append(append([]string{}, factoryArgs...), "--check")} {
		result(t, invoke(t, base, command...), 0)
	}
	if !bytes.Contains(read(t, filepath.Join(factoryOutput, "org/example/dedup/models/StatusDto.kt")), []byte("StatusDto default")) {
		t.Fatal("independent lexical component factory missing")
	}
	put(t, input, original)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("independent root factory changed original consumers or lock")
	}
	// This original root-bounds input now generates under a fresh lock. Reusing
	// the original unconstrained lock above must still report the shape change.
	put(t, input, bytes.ReplaceAll(original, []byte(`"enum":["on","off"]`), []byte(`"enum":["on","off"],"minLength":1`)))
	fresh := lockArgs(input, filepath.Join(base, "fresh"))
	result(t, invoke(t, base, fresh...), 0)
	result(t, invoke(t, base, append(fresh, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("fresh checked-root output changed the old locked inventory")
	}
}
