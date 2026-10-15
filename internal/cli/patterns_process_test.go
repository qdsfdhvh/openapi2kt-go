package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPatternFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "patterns.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, replacement := range []string{`"pattern":"["`, `"pattern":"(?<=a)b"`} {
		bad := bytes.Replace(input, []byte(`"pattern":"^[A-Z]{2}[0-9]+$"`), []byte(replacement), 1)
		if bytes.Equal(bad, input) {
			t.Fatal("injection unchanged")
		}
		put(t, p.Input, bad)
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "PatternsApi"}} {
			r := result(t, p.generate(t, args...), 1)
			if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != "/components/schemas/Code/pattern" {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid pattern changed output")
			}
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("missing validation flag changed output")
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	r := result(t, p.generate(t, "--validation", "--exclude-api", "PatternsApi"), 0)
	if len(r.Files) != 5 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	if !bytes.Contains(read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Code.kt")), []byte(".toRegex().containsMatchIn")) {
		t.Fatal("excluded API dropped checks")
	}
	result(t, p.generate(t, "--validation", "--exclude-api", "PatternsApi", "--check"), 0)
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--exclude-api", "PatternsApi"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op touched constrained models")
	}
}
