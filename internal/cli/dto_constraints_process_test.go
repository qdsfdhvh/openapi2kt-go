package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDTOValidationFlag(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "dto-constraints.json"))
	put(t, p.Input, input)
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	r := result(t, p.generate(t, "--validation"), 0)
	if len(r.Files) != 9 {
		t.Fatal(r)
	}
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 0)
	fails(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json"), 1, "E_SCHEMA_UNSUPPORTED")
	for _, args := range [][]string{nil, {"--check"}} {
		fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("disabling validation changed old output")
		}
	}
	for _, args := range [][]string{{"--validation=true"}, {"--validation=false"}, {"--validation", "false"}, {"--validation", "--validation"}} {
		fails(t, p.generate(t, args...), 2, "E_USAGE")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid flag changed output")
		}
	}
	bad := bytes.Replace(input, []byte(`"minLength":1,"maxLength":2`), []byte(`"minLength":3,"maxLength":2`), 1)
	if bytes.Equal(input, bad) {
		t.Fatal("bad bounds injection unchanged")
	}
	put(t, p.Input, bad)
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}, {"--validation", "--exclude-api", "ModelsApi"}} {
		fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid bounds changed output")
		}
	}
	fails(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--diagnostic-format", "json"), 1, "E_SCHEMA_INVALID")
	put(t, p.Input, input)
	r = result(t, p.generate(t, "--validation", "--exclude-api", "ModelsApi"), 0)
	if len(r.Files) != 3 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	if !bytes.Contains(read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Plain.kt")), []byte("init {")) {
		t.Fatal("exclusion dropped checks")
	}
	result(t, p.generate(t, "--validation", "--exclude-api", "ModelsApi", "--check"), 0)
	for _, command := range []string{"generate", "validate"} {
		h := invoke(t, "", command, "--help")
		if h.Code != 0 || !strings.Contains(string(h.Stdout), "--validation") {
			t.Fatal(h)
		}
	}
}

func TestUnboundedDTOValidationNoOp(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "basic.json")))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("opt-in flag rewrote unbounded output")
	}
}
