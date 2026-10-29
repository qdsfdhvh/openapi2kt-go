package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootDictionaryAllOfNativePreservation(t *testing.T) {
	checkRootDictionaryAllOfNativePreservation(t, `{"$ref":"#/components/schemas/ZMap"}`)
}

func TestRootDictionaryAllOfNeutralMembersNativePreservation(t *testing.T) {
	checkRootDictionaryAllOfNativePreservation(t, `{"description":"note"},{"$ref":"#/components/schemas/ZMap"},{}`)
}

func TestRootDictionaryAllOfOwnDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A/~":{"allOf":[{},{"$ref":"#/components/schemas/ZMap"}],"default":{"outer":"yes"}},"ZMap":{"additionalProperties":{"type":"string"},"default":{"target":"yes"}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, validate...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, flags...), 0)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("default wrapper regeneration changed output", checked)
		}
		for _, tc := range []struct{ original, replacement, pointer string }{
			{`"default":{"outer":"yes"}`, `"default":{"outer":false}`, "/components/schemas/A~1~0/default/outer"},
			{`"default":{"target":"yes"}`, `"default":{"target":false}`, "/components/schemas/ZMap/default/target"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.original), []byte(tc.replacement), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid independent default changed output or location", checked, check, failure)
				}
			}
			failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid independent default validation changed output or location", checked, failure)
			}
		}
	}
}

func checkRootDictionaryAllOfNativePreservation(t *testing.T, members string) {
	t.Helper()
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A/~":{"allOf":[` + members + `]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/A~1~0"}}},"ZMap":{"additionalProperties":{"type":"string"},"default":{"key":"ok"}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, validate...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, flags...), 0)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("deterministic regeneration changed output", checked)
		}
		for _, tc := range []struct{ original, replacement, code, pointer string }{
			{`"#/components/schemas/ZMap"`, `"#/components/schemas/Missing"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/A~1~0/allOf"},
			{`"default":{"key":"ok"}`, `"default":{"key":false}`, "E_SCHEMA_INVALID", "/components/schemas/ZMap/default/key"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.original), []byte(tc.replacement), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, tc.code)
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid wrapper changed output or source location", checked, check, failure)
				}
			}
			failure := fails(t, invoke(t, p.Root, validate...), 1, tc.code)
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid wrapper validation changed output or source location", checked, failure)
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}
