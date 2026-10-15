//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestCLIMetricsBusyRoot(t *testing.T) {
	p := warningProject(t)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	lock, err := os.OpenFile(filepath.Join(p.Output, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	measured(t, p.generate(t, "--metrics"), 4, "read", "generate", "commit")
	measured(t, p.generate(t, "--metrics", "--check"), 0, "read", "generate", "check")
	measured(t, p.generate(t, "--metrics", "--deny-warnings"), 1, "read", "generate", "warning_policy")
	measured(t, p.generate(t, "--metrics", "--check", "--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION", "--allow-warning=W_JSON_ELEMENT_APPROXIMATION"), 0, "read", "generate", "warning_policy", "check")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("metrics mutated locked/noop output")
	}
}
