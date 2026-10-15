package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNumericConstraintsFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "numeric-constraints.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ old, next, pointer string }{
		{`"multipleOf":2`, `"multipleOf":0`, "/components/schemas/Even/multipleOf"},
		{`"exclusiveMaximum":9223372036854775807`, `"exclusiveMaximum":true`, "/components/schemas/Even/exclusiveMaximum"},
	} {
		put(t, p.Input, bytes.Replace(input, []byte(tc.old), []byte(tc.next), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "NumbersApi"}} {
			r := result(t, p.generate(t, args...), 1)
			if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid numeric constraints changed old output")
			}
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("omitted opt-in changed old output")
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	r := result(t, p.generate(t, "--validation", "--exclude-api", "NumbersApi"), 0)
	if len(r.Files) != 4 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Record.kt"))
	if !bytes.Contains(text, []byte("%")) || !bytes.Contains(text, []byte("exclusiveMinimum")) || bytes.Contains(text, []byte("kotlin.Long.MIN_VALUE")) {
		t.Fatal("lost checks or shadow-safe literals", string(text))
	}
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--exclude-api", "NumbersApi", "--check"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "NumbersApi"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op touched numeric output")
	}
}
