package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDirectNullableComponentCountsNativeAdmissionAndPreservation(t *testing.T) {
	for _, tc := range []struct{ version, root string }{
		{"3.0.3", `"type":"object","nullable":true`},
		{"3.1.0", `"type":["object","null"]`},
	} {
		p := newProject(t)
		control := []byte(`{"openapi":"` + tc.version + `","components":{"schemas":{"Record":{` + tc.root + `,"properties":{"value":{"type":"string"}}}}}}`)
		put(t, p.Input, control)
		result(t, p.generate(t), 0)
		put(t, filepath.Join(p.Output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, p.Output)
		counted := bytes.Replace(control, []byte(`,"properties"`), []byte(`,"minProperties":1,"maxProperties":2,"properties"`), 1)
		put(t, p.Input, counted)
		result(t, p.generate(t, "--validation", "--check"), 3)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("candidate check wrote old output")
		}
		result(t, p.generate(t, "--validation"), 0)
		text := read(t, filepath.Join(p.Output, "org/example/catalog/models/Record.kt"))
		if !bytes.Contains(text, []byte("generatedSerializer()")) || !bytes.Contains(text, []byte("element.size < 1")) || !bytes.Contains(text, []byte("element.size > 2")) {
			t.Fatal(string(text))
		}
		before = snapshot(t, p.Output)
		result(t, p.generate(t, "--validation", "--check"), 0)
		result(t, p.generate(t, "--validation"), 0)
		result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("native check/noop/validate changed bytes/modes/mtimes")
		}
		for _, bad := range []struct {
			input    []byte
			code, at string
		}{
			{counted, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/minProperties"},
			{bytes.Replace(counted, []byte(`"minProperties":1`), []byte(`"minProperties":-1`), 1), "E_SCHEMA_INVALID", "/components/schemas/Record/minProperties"},
			{bytes.Replace(counted, []byte(`"maxProperties":2`), []byte(`"maxProperties":0`), 1), "E_SCHEMA_INVALID", "/components/schemas/Record/maxProperties"},
			{bytes.Replace(counted, []byte(`"type":"string"`), []byte(`"type":"string","default":false`), 1), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/default"},
		} {
			put(t, p.Input, bad.input)
			flags := []string{}
			if bad.code != "E_SCHEMA_UNSUPPORTED" {
				flags = append(flags, "--validation")
			}
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failed := fails(t, p.generate(t, args...), 1, bad.code)
				if *failed.Diagnostics[0].Pointer != bad.at || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal(bad, failed)
				}
			}
		}
	}
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/direct-nullable-component-counts.json")))
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto"), 0)
	result(t, p.generate(t, "--validation", "--model-name-suffix", "Dto", "--check"), 0)
}

func directNullableCountLockInput(t testing.TB, base string) string {
	t.Helper()
	path := filepath.Join(base, "input.json")
	counted := `{"type":["object","null"],"minProperties":0,"maxProperties":1,"properties":{"value":{"type":"string"}}}`
	model := `{"properties":{"direct":{"$ref":"#/components/schemas/Record"},"owned":` + counted + `}}`
	op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":` + model + `}}}}}}`
	put(t, path, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":`+counted+`}},"paths":{"/alpha":`+op+`,"/beta":`+strings.ReplaceAll(op, `"alpha"`, `"beta"`)+`}}`))
	return path
}

func TestDirectNullableComponentCountsNativeCoreLockAndAuthority(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		base := t.TempDir()
		input := directNullableCountLockInput(t, base)
		output := filepath.Join(base, "out")
		original := read(t, input)
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
				t.Fatal("actual core/native source differs", file.Path)
			}
		}
		lock := filepath.Join(output, ".openapi-kt-names.json")
		if !bytes.Equal(got.Artifacts.NameLock, read(t, lock)) {
			t.Fatal("actual core/native lock differs")
		}
		put(t, filepath.Join(output, "handwritten.kt"), []byte("keep"))
		before := snapshot(t, output)
		result(t, invoke(t, base, args...), 0)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("native replay changed manifest/raw lock/modes/mtime")
		}
		for _, tc := range []struct{ replacement, code string }{{`"maxProperties":2`, "E_NAME_LOCK_CHANGED"}, {`"maxProperties":-1`, "E_SCHEMA_INVALID"}} {
			put(t, input, bytes.ReplaceAll(original, []byte(`"maxProperties":1`), []byte(tc.replacement)))
			fails(t, invoke(t, base, args...), 1, tc.code)
			if !reflect.DeepEqual(before, snapshot(t, output)) {
				t.Fatal("failure replaced owned output", tc)
			}
		}
		put(t, input, original)
		raw := read(t, lock)
		put(t, lock, append(raw, ' '))
		modified := snapshot(t, output)
		fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(modified, snapshot(t, output)) {
			t.Fatal("modified raw lock overwritten")
		}
		// Direct output mode still cannot write a named/locked owner root using
		// another input or reset the recorded authority.
		fails(t, invoke(t, base, "generate", "--input", input, "--output", output, "--package-name", "org.example.dedup", "--validation", "--diagnostic-format", "json"), 4, "E_MANIFEST_VERSION")
	}
}
