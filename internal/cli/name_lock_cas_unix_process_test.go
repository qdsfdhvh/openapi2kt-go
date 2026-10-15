//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestNameLockNativeStaleSnapshotCASWithFIFOBarrier(t *testing.T) {
	for _, check := range []bool{false, true} {
		base := t.TempDir()
		input := lockInput(t, base)
		root := filepath.Join(base, "out")
		args := lockArgs(input, root)
		result(t, invoke(t, base, args...), 0)
		original := read(t, input)
		fifo := filepath.Join(base, "barrier.json")
		if e := syscall.Mkfifo(fifo, 0600); e != nil {
			t.Fatal(e)
		}
		blocked := append([]string(nil), args...)
		for i, a := range blocked {
			if a == "--input" {
				blocked[i+1] = fifo
			}
		}
		if check {
			blocked = append(blocked, "--check")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, blocked...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
		defer func() { _ = cmd.Process.Kill() }()
		// Opening the write side completes only once CLI has finished readonly
		// state load and opened document input. No shell polling/sleeps or CLI hook.
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
			t.Fatal("FIFO state-read barrier timed out")
		}
		lockGrow(t, input)
		result(t, invoke(t, base, args...), 0)
		competitor := snapshot(t, root)
		if _, e := writer.Write(original); e != nil {
			t.Fatal(e)
		}
		_ = writer.Close()
		e := cmd.Wait()
		exit := 0
		if e != nil {
			if x, ok := e.(*exec.ExitError); ok {
				exit = x.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_NAME_LOCK_STALE")
		if !reflect.DeepEqual(competitor, snapshot(t, root)) {
			t.Fatal("native stale write/check replaced competitor state")
		}
	}
}
