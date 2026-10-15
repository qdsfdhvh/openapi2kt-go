//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestConfigurationAdoptionBusyCheckMetadataOnly(t *testing.T) {
	_, cfg, root := legacyConfigRoot(t)
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
	args := []string{"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--diagnostic-format=json"}
	result(t, invoke(t, repo, append(args, "--check")...), 3)
	fails(t, invoke(t, repo, args...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("locked/check adoption changed old manifest")
	}
}
