package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExplicitNullLiteralsNativeCheckAndSourceFailurePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/explicit-null-literals.json"))
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Null check changed output")
		}
		changed := bytes.Replace(original, []byte("\"enum\": [\n          null\n        ]"), []byte("\"enum\": [\n          null,\n          null\n        ]"), 1)
		if bytes.Equal(changed, original) {
			t.Fatal("Missing original null vocabulary")
		}
		put(t, p.Input, changed)
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/NullVocabulary/enum/1" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Original null index failure changed output", failure)
		}
	}
}
