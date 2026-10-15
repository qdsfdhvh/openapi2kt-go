package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrimitiveMapDefaultsNativeReadOnlyAndOriginalMemberFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/primitive-map-defaults.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Map check changed bytes/mtime")
		}
		for _, tc := range []struct {
			before, after, pointer string
		}{
			{"-9223372036854775808", "-9223372036854775809", "/components/schemas/Holder/properties/longs/default/min"},
			{"2024-02-29", "2023-02-29", "/components/schemas/Holder/properties/dates/default/a~1~0"},
			{"2024-02-29T12:34:56.123456789+02:30", "2024-02-29T12:34:56.1234567891+02:30", "/components/schemas/Holder/properties/times/default/stamp"},
		} {
			changed := bytes.Replace(original, []byte(tc.before), []byte(tc.after), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing original member", tc.before)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Bad original member changed owned outputs", tc, failure)
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored map check mutated existing state")
		}
	}
}
