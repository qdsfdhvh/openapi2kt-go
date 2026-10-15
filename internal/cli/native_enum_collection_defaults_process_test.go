package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNativeEnumCollectionDefaultsNativeFailurePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/native-enum-collection-defaults.json"))
		var flags []string
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Check changed native enum collection outputs")
		}
		for _, tc := range []struct{ old, replacement, pointer string }{
			{`"default": [1, null, 2]`, `"default": [1, null, 3]`, "/components/schemas/Holder/properties/codes/default/2"},
			{`"default": {"a/~": true, "$ref": true}`, `"default": {"a/~": false, "$ref": true}`, "/components/schemas/Holder/properties/flags/default/a~1~0"},
			{`"default": {"half": 0.5}`, `"default": {"half": 0.500000001}`, "/components/schemas/Holder/properties/floats/default/half"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing original member", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Invalid enum default changed existing owned output", tc, failure)
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored enum default check mutated output")
		}
	}
}
