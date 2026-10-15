package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNestedBuiltinCollectionDefaultsNativeFailurePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/nested-builtin-collection-defaults.json"))
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--deny-warnings")...), 0)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, flags...), 0)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		validated := append([]string{"validate", "--input", p.Input, "--package-name", "org.example.catalog", "--diagnostic-format", "json"}, flags...)
		result(t, invoke(t, "", validated...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("noop/check/validate changed bytes or mtime")
		}
		for _, tc := range []struct{ old, replacement, pointer string }{
			{"-9223372036854775808", "-9223372036854775809", "/components/schemas/Holder/properties/rows/default/0/0"},
			{`"a/~": "quote\"$😀"`, `"a/~": 1`, "/components/schemas/Holder/properties/listMaps/default/0/a~1~0"},
			{`"default": [[1]]`, `"default": [[false]]`, "/components/schemas/Holder/properties/required/default/0/0"},
			{`"default": {"a": ["yes"]}`, `"default": {"a": [false]}`, "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/labels/default/a/0"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("missing replacement", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid annotation changed output", tc, failure)
			}
		}
	}
}
