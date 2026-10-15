//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestWarningPolicyBusyRoot(t *testing.T) {
	p := warningProject(t)
	result(t, p.generate(t), 0)
	lock, err := os.OpenFile(filepath.Join(p.Output, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	before := snapshot(t, p.Output)
	warningDenied(t, p.generate(t, "--deny-warnings"))
	warningDenied(t, p.generate(t, "--deny-warnings", "--check"))
	all := []string{"--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION", "--allow-warning=W_JSON_ELEMENT_APPROXIMATION"}
	result(t, p.generate(t, append(all, "--check")...), 0)
	fails(t, p.generate(t, all...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("warning policy/lock/check changed files")
	}
}
