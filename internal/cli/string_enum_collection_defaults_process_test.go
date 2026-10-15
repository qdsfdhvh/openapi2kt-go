package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStringEnumCollectionDefaultsNativeFailurePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/string-enum-collection-defaults.json"))
		var flags []string
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		for _, tc := range []struct{ old, replacement, pointer string }{
			{`"default": ["one", null, "two"]`, `"default": ["one", null, "future"]`, "/components/schemas/Holder/properties/codes/default/2"},
			{`"default": {"a/~": "yes", "$ref": "no"}`, `"default": {"a/~": "future", "$ref": "no"}`, "/components/schemas/Holder/properties/labels/default/a~1~0"},
			{`"default": ["one"]`, `"default": ["future"]`, "/components/schemas/Holder/properties/required/default/0"},
			{`"codes": {"type": "array", "nullable": true, "items": {"type": "string", "enum": ["one", "two"]}, "default": ["one"]}`, `"codes": {"type": "array", "nullable": true, "items": {"type": "string", "enum": ["one", "two"]}, "default": ["future"]}`, "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/codes/default/0"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing replacement", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Bad annotation changed owned output", tc, failure)
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored check mutated output")
		}
	}
}
