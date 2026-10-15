//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestConfigMetricsBusySharedRoot(t *testing.T) {
	cfg, root := runtimeConfiguration(t, true)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
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
	args := []string{"generate", "--config", cfg, "--metrics", "--diagnostic-format=json"}
	for _, check := range []bool{false, true} {
		a := append([]string{}, args...)
		phase := "commit"
		if check {
			a = append(a, "--check")
			phase = "check"
		}
		out := invoke(t, repo, a...)
		if check {
			result(t, out, 0)
		} else {
			fails(t, out, 4, "E_OUTPUT_BUSY")
		}
		m := configMetrics(t, out)
		if m.PlannedFiles == 0 || m.PlannedBytes == 0 {
			t.Fatal("generated but not written counts missing", m)
		}
		if _, ok := m.PhaseNS[phase]; !ok {
			t.Fatal("missing actual batch phase", m)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("busy metrics/check changed old tree")
		}
	}
}
