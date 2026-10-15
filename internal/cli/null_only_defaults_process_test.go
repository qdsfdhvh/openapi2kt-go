package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNullOnlyDefaultsNativeCheckAndFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/null-only-defaults.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Null default check changed output bytes or mtimes")
		}
		changed := bytes.Replace(original, []byte("\"enum\": [\n          null\n        ]"), []byte("\"enum\": [\n          null,\n          null\n        ]"), 1)
		if bytes.Equal(changed, original) {
			t.Fatal("Missing original inferred null vocabulary")
		}
		put(t, p.Input, changed)
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/NullVocabulary/enum/1" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Null default source index failure changed existing output", failure)
		}
		put(t, p.Input, original)
		invalidDefault := bytes.Replace(original, []byte(`"default": null`), []byte(`"default": false`), 1)
		if bytes.Equal(invalidDefault, original) {
			t.Fatal("Missing original null default annotation")
		}
		put(t, p.Input, invalidDefault)
		defaultFailure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if defaultFailure.Diagnostics[0].Pointer == nil || *defaultFailure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/requiredNull/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Invalid null default changed existing output", defaultFailure)
		}
	}
}
