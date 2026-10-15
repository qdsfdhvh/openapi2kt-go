package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectionCountsFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "collection-counts.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ old, next, pointer string }{
		{`"minItems": 1`, `"minItems": -1`, "/components/schemas/Bag/additionalProperties/minItems"},
		{`"minProperties": 1`, `"minProperties": 3`, "/components/schemas/Bag/maxProperties"},
	} {
		put(t, p.Input, bytes.Replace(input, []byte(tc.old), []byte(tc.next), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "CountsApi"}} {
			r := result(t, p.generate(t, args...), 1)
			if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid counts changed prior output")
			}
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("missing opt-in changed output")
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	r := result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 0)
	if len(r.Files) != 3 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Record.kt"))
	if !bytes.Contains(text, []byte("minItems")) || !bytes.Contains(text, []byte("minProperties")) {
		t.Fatal("lost model checks", string(text))
	}
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi", "--check"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "CountsApi"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("count no-op touched output")
	}
}
