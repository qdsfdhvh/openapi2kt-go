package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStructuredConstFieldsNativeCheckAndFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/structured-const-fields.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		good := result(t, p.generate(t, flags...), 0)
		found := false
		for _, d := range good.Warnings {
			found = found || d.Code == "W_CONST_STRUCTURED_APPROXIMATION" && d.Pointer != nil && *d.Pointer == "/components/schemas/Holder/properties/payload/const"
		}
		if !found {
			t.Fatal("Native generation omitted structured const approximation warning", good)
		}
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Structured const readonly check changed output bytes or mtimes")
		}
		for _, tc := range []struct {
			input         []byte
			code, pointer string
		}{
			{bytes.Replace(original, []byte(`"type": "object", "const": {"$ref"`), []byte(`"type": "array", "const": {"$ref"`), 1), "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/payload/const"},
			{bytes.Replace(original, []byte(`"payload": {"type": "object", "const":`), []byte(`"payload": {"default": {}, "type": "object", "const":`), 1), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/payload/default"},
		} {
			if bytes.Equal(tc.input, original) {
				t.Fatal("Missing original structured const fixture vocabulary")
			}
			put(t, p.Input, tc.input)
			failure := fails(t, p.generate(t, flags...), 1, tc.code)
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Structured const source failure changed existing output", failure)
			}
		}
	}
}
