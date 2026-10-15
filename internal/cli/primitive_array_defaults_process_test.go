package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrimitiveArrayDefaultsNativeReadOnlyAndOriginalMemberFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/primitive-array-defaults.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Array check changed bytes/mtime")
		}
		for _, tc := range []struct{ old, bad, pointer string }{
			{"-9223372036854775808", "-9223372036854775809", "/components/schemas/Holder/properties/longs/default/0"},
			{"2024-02-29", "2023-02-29", "/components/schemas/Holder/properties/dates/default/0"},
			{"2024-02-29T12:34:56.123456789+02:30", "2024-02-29T12:34:56.1234567891Z", "/components/schemas/Holder/properties/times/default/0"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.bad), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing original array member", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Bad original member changed owned outputs", failure)
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored array check mutated existing state")
		}
	}
}
