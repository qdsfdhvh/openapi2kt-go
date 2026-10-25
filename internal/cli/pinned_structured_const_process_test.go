package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPinnedStructuredConstNativeSourceAndFailurePreservation(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			original := read(t, filepath.Join(repo, "tests/fixtures/http/pinned-structured-const.json"))
			put(t, p.Input, original)
			flags := []string{"--model-style", style}
			if checked {
				flags = append(flags, "--validation")
			}
			good := result(t, p.generate(t, flags...), 0)
			found := false
			for _, d := range good.Warnings {
				found = found || d.Code == "W_CONST_STRUCTURED_APPROXIMATION" && d.Pointer != nil && *d.Pointer == "/components/schemas/Choice/oneOf/0/properties/payload/const"
			}
			if !found {
				t.Fatal("Original pinned source warning missing", good)
			}
			before := snapshot(t, p.Output)
			result(t, p.generate(t, append(flags, "--check")...), 0)
			result(t, p.generate(t, flags...), 0)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Pinned no-op/check changed bytes or attrs")
			}
			put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{},"default":{}}}},{"properties":{"tag":{"const":"two"}}}]}}}}`))
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Choice/oneOf/0/properties/payload/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Pinned source error rewrote old output", failure)
			}
		}
	}
}
