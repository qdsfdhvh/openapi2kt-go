//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestConfigurationBusyRootAndReadOnlyCheck(t *testing.T) {
	dir, cfg := configFiles(t)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	root := filepath.Join(dir, "generated")
	lock, e := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if e := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	before := snapshot(t, root)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("locked config changed outputs")
	}
}
