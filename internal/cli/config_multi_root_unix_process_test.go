//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestConfigurationMultiRootBusyPeerReleasesLocks(t *testing.T) {
	dir, cfg := multiRootConfig(t)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	a, b := filepath.Join(dir, "generated"), filepath.Join(dir, "generated-beta")
	lock, e := os.OpenFile(filepath.Join(b, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if e := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	beforeA, beforeB := snapshot(t, a), snapshot(t, b)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("busy peer changed batch")
	}
	first, e := os.OpenFile(filepath.Join(a, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	if e := syscall.Flock(int(first.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal("batch did not release first root lock", e)
	}
	_ = syscall.Flock(int(first.Fd()), syscall.LOCK_UN)
}
