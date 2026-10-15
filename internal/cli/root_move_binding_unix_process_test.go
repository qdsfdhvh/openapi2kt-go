//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveNativeFullConfigCanonicalAncestorBinding(t *testing.T) {
	base := t.TempDir()
	names := []string{"alpha", "beta"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, true, false, true)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	src, dst := filepath.Join(base, "generated"), filepath.Join(base, "new")
	alias := filepath.Join(base, "alias")
	if e := os.Symlink(base, alias); e != nil {
		t.Fatal(e)
	}
	put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "output_dir='generated'", "output_dir='alias/generated'")))
	run := append(append([]string(nil), args...), "--move-output-from", src, "--output", dst)
	before := snapshot(t, src)
	result(t, invoke(t, base, append(run, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("alias preflight changed source")
	}
	result(t, invoke(t, base, run...), 0)
	after := snapshot(t, dst)
	for path, old := range before {
		if path != ".openapi-kt.lock" && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("canonical binding lost source", path)
		}
	}
}

func TestRootMoveNativeDisjointAndUnrelatedConfigFailBeforeInput(t *testing.T) {
	for _, mode := range []string{"disjoint", "unrelated"} {
		base := t.TempDir()
		names := []string{"alpha", "beta"}
		lifecycleConfigInputs(t, base, names)
		cfg := lifecycleConfig(t, base, names, true, mode == "disjoint", false)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		if e := os.Remove(filepath.Join(base, "alpha.json")); e != nil {
			t.Fatal(e)
		}
		source := filepath.Join(base, "alpha-output")
		if mode == "unrelated" {
			source = filepath.Join(base, "other-source")
		}
		run := append(args, "--move-output-from", source, "--output", filepath.Join(base, "new"))
		fails(t, invoke(t, base, run...), 4, "E_OUTPUT_ROOT_MOVE_INVALID")
	}
}

func TestRootMoveNativeReadonlyAbsentTargetAndStdin(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	dst := filepath.Join(p.Root, "missing", "target")
	args := []string{"generate", "--input", p.Input, "--output", dst, "--move-output-from", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json"}
	before := snapshot(t, p.Output)
	result(t, invoke(t, p.Root, append(args, "--check")...), 3)
	result(t, invoke(t, p.Root, lifecycleValidate(args)...), 0)
	if _, e := os.Stat(filepath.Dir(dst)); !os.IsNotExist(e) {
		t.Fatal("readonly created target ancestor")
	}
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("readonly changed source")
	}
	// Same stream adapter and EOF boundary are retained for direct root move.
	stdinArgs := append([]string(nil), args...)
	stdinArgs[2] = "-"
	out := invokeInput(t, p.Root, bytes.NewReader(read(t, p.Input)), stdinArgs...)
	result(t, out, 0)
}
