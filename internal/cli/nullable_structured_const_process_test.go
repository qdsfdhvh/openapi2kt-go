package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullableStructuredConstNativeFailureAndReadOnlyPreserveExistingOutput(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/nullable-structured-const.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Nullable JSON root check changed files/mtimes")
	}
	for _, tc := range []struct {
		input         []byte
		code, pointer string
	}{
		{bytes.Replace(input, []byte(`"Array": {"type": "array", "nullable": true, "const":`), []byte(`"Array": {"type": "array", "default": null, "nullable": true, "const":`), 1), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Array/default"},
		{bytes.Replace(input, []byte(`"const": {"n": 1}`), []byte(`"const": []`), 1), "E_SCHEMA_INVALID", "/components/schemas/Object/const"},
	} {
		if bytes.Equal(input, tc.input) {
			t.Fatal("Missing original nullable literal fixture")
		}
		put(t, p.Input, tc.input)
		bad := fails(t, p.generate(t, "--validation"), 1, tc.code)
		if bad.Diagnostics[0].Pointer == nil || *bad.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Nullable original-source failure changed owned output", bad)
		}
	}
}
