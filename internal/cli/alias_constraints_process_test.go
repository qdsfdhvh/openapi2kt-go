package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAliasConstraintsFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "alias-constraints.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	for _, replacement := range []string{`"minLength":-1`, `"minLength":3`} {
		bad := bytes.Replace(input, []byte(`"minLength":1`), []byte(replacement), 1)
		put(t, p.Input, bad)
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "AliasesApi"}} {
			r := result(t, p.generate(t, args...), 1)
			if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid alias changed old output")
			}
		}
	}
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("omitted opt-in changed output")
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation", "--exclude-api", "AliasesApi"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("removed unowned file")
	}
	text := read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Record.kt"))
	if !bytes.Contains(text, []byte("this.counts.values.forEach")) || !bytes.Contains(text, []byte("require(value >= -2L)")) {
		t.Fatal("excluded operations erased model checks", string(text))
	}
	result(t, p.generate(t, "--validation", "--exclude-api", "AliasesApi", "--check"), 0)
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--exclude-api", "AliasesApi"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("no-op touched output")
	}
}
