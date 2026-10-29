package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestReferencedStringEnumNullMemberDefaultsNativeLockAndPreservation(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := filepath.Join(base, "input.json")
		output := filepath.Join(base, "out")
		original := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":["string","null"],"enum":[null,"on","off","null"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Status"}}}},"paths":{"/a":{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"scalar":{"$ref":"#/components/schemas/Status","default":"on"},"a/~":{"$ref":"#/components/schemas/Rows","default":[null,"on"]}}}}}}}}},"/b":{"get":{"operationId":"beta","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"scalar":{"$ref":"#/components/schemas/Status","default":"on"},"a/~":{"$ref":"#/components/schemas/Rows","default":[null,"on"]}}}}}}}}}}}`)
		put(t, input, original)
		args := append(lockArgs(input, output), "--model-style="+string(style))
		validate := append([]string{}, args...)
		validate[0] = "validate"
		result(t, invoke(t, base, append(args, "--check")...), 3)
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: original, Options: core.GenerationOptions{PackageName: "org.example.dedup", ModelNameSuffix: "Dto", ModelStyle: style, Validation: true, HoistSharedShapes: true, LockHoistedNames: true, DeduplicateModels: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range report.Artifacts.Files {
			if string(read(t, filepath.Join(output, file.Path))) != file.Content {
				t.Fatal("native/core parity", file.Path)
			}
		}
		if !bytes.Equal(report.Artifacts.NameLock, read(t, filepath.Join(output, ".openapi-kt-names.json"))) {
			t.Fatal("raw lock parity")
		}
		put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, output)
		for _, tc := range []struct{ old, replacement, code string }{
			{`"default":"on"`, `"default":"off"`, "E_NAME_LOCK_CHANGED"},
			{`"default":[null,"on"]`, `"default":["on",null]`, "E_NAME_LOCK_CHANGED"},
			{`"default":[null,"on"]`, `"default":[null,"future"]`, "E_SCHEMA_INVALID"},
			{`"default":"on"`, `"default":"future"`, "E_SCHEMA_INVALID"},
			{`"enum":[null,"on","off","null"]`, `"enum":[null,"on",null,"off","null"]`, "E_SCHEMA_INVALID"},
			{`"enum":[null,"on","off","null"]`, `"enum":[null,"on","off","null"],"default":"future"`, "E_SCHEMA_INVALID"},
		} {
			put(t, input, bytes.ReplaceAll(original, []byte(tc.old), []byte(tc.replacement)))
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
				failed := fails(t, invoke(t, base, command...), 1, tc.code)
				if tc.replacement == `"default":[null,"future"]` && (failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/paths/~1a/get/responses/200/content/application~1json/schema/properties/a~1~0/default/1") {
					t.Fatal("escaped original member pointer", failed)
				}
			}
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("failed candidate changed old output", tc)
			}
		}
		put(t, input, bytes.ReplaceAll(original, []byte(`"default":"on"`), []byte(`"default":"\u006fn"`)))
		result(t, invoke(t, base, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("equivalent String spelling changed identity")
		}
		// The original null factory refusal now composes with consuming defaults.
		put(t, input, bytes.ReplaceAll(original, []byte(`"enum":[null,"on","off","null"]`), []byte(`"enum":[null,"on","off","null"],"default":null`)))
		factoryOutput := filepath.Join(base, "factory-out")
		factoryArgs := lockArgs(input, factoryOutput)
		factoryArgs = append(factoryArgs, "--model-style="+string(style))
		factoryValidate := append([]string{}, factoryArgs...)
		factoryValidate[0] = "validate"
		for _, command := range [][]string{factoryValidate, factoryArgs, append(append([]string{}, factoryArgs...), "--check")} {
			result(t, invoke(t, base, command...), 0)
		}
		if !bytes.Contains(read(t, filepath.Join(factoryOutput, "org/example/dedup/models/StatusDto.kt")), []byte("StatusDto default")) {
			t.Fatal("independent nullable component factory missing")
		}
		put(t, input, original)
		result(t, invoke(t, base, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("independent null factory changed original consumers or lock")
		}
	}
}
