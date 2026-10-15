package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckedArrayDefaultsNativeFailurePreservesOutput(t *testing.T) {
	p := newProject(t)
	original := read(t, filepath.Join(repo, "tests/fixtures/http/checked-array-defaults.json"))
	put(t, p.Input, original)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default": [2, null]`, `"default": [2, 2]`, "/components/schemas/Holder/properties/values/default/1"},
		{`"default": [2, null]`, `"default": []`, "/components/schemas/Holder/properties/values/default"},
		{`"default": [2, null]`, `"default": [3]`, "/components/schemas/Holder/properties/values/default/0"},
	} {
		changed := bytes.Replace(original, []byte(tc.from), []byte(tc.to), 1)
		if bytes.Equal(changed, original) {
			t.Fatal("Missing original default")
		}
		put(t, p.Input, changed)
		for _, flags := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Invalid checked default changed output", failure)
			}
		}
	}
	put(t, p.Input, original)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Check/default-off changed existing output")
	}
}
