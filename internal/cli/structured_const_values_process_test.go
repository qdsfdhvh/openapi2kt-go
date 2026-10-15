package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStructuredConstValuesNativeReadOnlyAndSourceFailurePreservation(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/structured-const-values.json"))
	put(t, p.Input, input)
	good := result(t, p.generate(t, "--validation"), 0)
	found := false
	for _, warning := range good.Warnings {
		found = found || warning.Code == "W_CONST_STRUCTURED_APPROXIMATION" && warning.Pointer != nil && *warning.Pointer == "/components/schemas/Array/const"
	}
	if !found {
		t.Fatal("Structured root warning lost original const source", good)
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Structured value check changed output bytes/mtime")
	}
	for _, tc := range []struct {
		input         []byte
		code, pointer string
	}{
		{bytes.Replace(input, []byte(`"Array": {"type": "array", "const":`), []byte(`"Array": {"type": "array", "default": [], "const":`), 1), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Array/default"},
		{bytes.Replace(input, []byte(`"Alias": {"$ref": "#/components/schemas/Object"}`), []byte(`"Alias": {"$ref": "#/components/schemas/Object", "nullable": true}`), 1), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Alias"},
	} {
		if bytes.Equal(input, tc.input) {
			t.Fatal("Missing original literal/alias fixture vocabulary")
		}
		put(t, p.Input, tc.input)
		if tc.pointer == "/components/schemas/Alias" {
			// Keep the original negative input verbatim as a nullable admission.
			result(t, p.generate(t, "--validation"), 0)
			alias := read(t, filepath.Join(p.Output, "org/example/catalog/models/Alias.kt"))
			if !bytes.Contains(alias, []byte("typealias Alias = kotlinx.serialization.json.JsonObject?")) {
				t.Fatal("Original nullable alias input lost its actual classifier")
			}
			accepted := snapshot(t, p.Output)
			result(t, p.generate(t, "--validation", "--check"), 0)
			if !reflect.DeepEqual(accepted, snapshot(t, p.Output)) {
				t.Fatal("Nullable alias admission check changed output")
			}
			put(t, p.Input, input)
			result(t, p.generate(t, "--validation"), 0)
			before = snapshot(t, p.Output)
			continue
		}
		failed := fails(t, p.generate(t, "--validation"), 1, tc.code)
		if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Structured value source failure changed existing output", failed)
		}
	}
}
