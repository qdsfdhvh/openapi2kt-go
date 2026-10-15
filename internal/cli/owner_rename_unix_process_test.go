//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOwnerRenameNativeUnixBusyReadonlyAndLockRelease(t *testing.T) {
	base := t.TempDir()
	names := []string{"alpha", "beta"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, true, true, false)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	renameConfig(t, cfg, "alpha", "gamma")
	renameConfig(t, cfg, "beta", "delta")
	run := append(append([]string(nil), args...), "--rename-owner=alpha:gamma", "--rename-owner=beta:delta")
	root := filepath.Join(base, "beta-output")
	f, e := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	before := snapshot(t, base)
	result(t, invoke(t, base, append(run, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, base)) {
		t.Fatal("busy write/check/validate changed output")
	}
	// The earlier root must have released its lock after the later busy peer.
	result(t, invoke(t, base, append(args, "--spec=gamma", "--rename-owner=alpha:gamma")...), 0)
}

func TestOwnerRenameNativeUnixFIFOExpectedStateCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		base := t.TempDir()
		names := []string{"alpha"}
		lifecycleConfigInputs(t, base, names)
		cfg := lifecycleConfig(t, base, names, true, false, false)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		competitorCfg := filepath.Join(base, "competitor.toml")
		put(t, competitorCfg, read(t, cfg))
		input := filepath.Join(base, "alpha.json")
		original := read(t, input)
		fifo := filepath.Join(base, "rename-barrier.json")
		if e := syscall.Mkfifo(fifo, 0600); e != nil {
			t.Fatal(e)
		}
		renameConfig(t, cfg, "alpha", "gamma")
		put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "alpha.json", "rename-barrier.json")))
		run := append(append([]string(nil), args...), "--spec=gamma", "--rename-owner=alpha:gamma")
		if mode == "check" {
			run = append(run, "--check")
		}
		if mode == "validate" {
			run = lifecycleValidate(run)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, run...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
		defer cmd.Process.Kill()
		opened := make(chan *os.File, 1)
		errs := make(chan error, 1)
		go func() {
			f, e := os.OpenFile(fifo, os.O_WRONLY, 0600)
			if e != nil {
				errs <- e
			} else {
				opened <- f
			}
		}()
		var writer *os.File
		select {
		case writer = <-opened:
		case e := <-errs:
			t.Fatal(e)
		case <-ctx.Done():
			t.Fatal("rename FIFO state-read barrier timed out")
		}
		defer writer.Close()
		lockGrow(t, input)
		result(t, invoke(t, base, "generate", "--config", competitorCfg, "--diagnostic-format=json"), 0)
		root := filepath.Join(base, "generated")
		before := snapshot(t, root)
		if _, e := writer.Write(original); e != nil {
			t.Fatal(e)
		}
		if e := writer.Close(); e != nil {
			t.Fatal(e)
		}
		e := cmd.Wait()
		code := 0
		if e != nil {
			if exit, ok := e.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		fails(t, invocation{Code: code, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_OWNER_RENAME_STALE")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("stale rename lost competing state")
		}
	}
}
