package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBytesFieldDefaultsNativeReadOnlyAndOriginalMemberFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/bytes-field-defaults.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Bytes default check changed bytes/mtime")
		}
		// Every rejected member reports its own element pointer and leaves the
		// previously generated outputs untouched.
		for _, tc := range []struct{ old, bad, pointer string }{
			{"127", "128", "/components/schemas/Holder/properties/requiredBytes/default/3"},
			{"-128", "-129", "/components/schemas/Holder/properties/requiredBytes/default/0"},
			{"\"binary\",\n            \"default\": [\n              4\n            ]", "\"binary\",\n            \"default\": \"AAEC\"", "/components/schemas/Holder/properties/optionalBytes/default"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.bad), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing original bytes member", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer {
				t.Fatalf("Bad bytes member reported %v, want %s", failure.Diagnostics[0].Pointer, tc.pointer)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Bad original bytes member changed owned outputs")
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored bytes check mutated existing state")
		}
	}
}
