//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestNameLockNativeBusyReadCheckValidateAndPolicy(t *testing.T) {
	for _, config := range []bool{false, true} {
		base := t.TempDir()
		input := lockInput(t, base)
		root := filepath.Join(base, "generated")
		args := lockArgs(input, root)
		if config {
			cfg := lockConfig(t, base, []string{"alpha"})
			args = []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		}
		result(t, invoke(t, base, args...), 0)
		f, e := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, root)
		result(t, invoke(t, base, append(args, "--check")...), 0)
		v := append([]string(nil), args...)
		v[0] = "validate"
		result(t, invoke(t, base, v...), 0)
		fails(t, invoke(t, base, args...), 4, "E_OUTPUT_BUSY")
		lockGrow(t, input)
		result(t, invoke(t, base, append(args, "--check")...), 3)
		fails(t, invoke(t, base, args...), 4, "E_OUTPUT_BUSY")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("heldlock/readcheckvalidate mutated")
		}
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
	// A warning-bearing document still reads lock state before core policy, but
	// policy failure must precede output acquisition/mutation (including busy).
	p := warningProject(t)
	args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.catalog", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	f, e := os.OpenFile(filepath.Join(p.Output, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	before := snapshot(t, p.Output)
	denied := result(t, invoke(t, repo, append(args, "--deny-warnings")...), 1)
	if len(denied.Diagnostics) == 0 || denied.Diagnostics[0].Code != "E_WARNING_DENIED" {
		t.Fatal(denied)
	}
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("policy wrote metadata")
	}
}
