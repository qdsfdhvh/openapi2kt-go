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

func TestNestedNativeEnumCollectionDefaultsNativeFIFOStaleCAS(t *testing.T) {
	for _, check := range []bool{false, true} {
		base := t.TempDir()
		input := nestedNativeEnumLockInput(t, base)
		root := filepath.Join(base, "out")
		args := lockArgs(input, root)
		result(t, invoke(t, base, args...), 0)
		put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
		original := read(t, input)
		fifo := filepath.Join(base, "barrier.json")
		if err := syscall.Mkfifo(fifo, 0600); err != nil {
			t.Fatal(err)
		}
		blocked := append([]string(nil), args...)
		for i, arg := range blocked {
			if arg == "--input" {
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
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		opened := make(chan *os.File, 1)
		errs := make(chan error, 1)
		go func() {
			f, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
			if err != nil {
				errs <- err
			} else {
				opened <- f
			}
		}()
		var writer *os.File
		select {
		case writer = <-opened:
		case err := <-errs:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal("FIFO state-read barrier timed out")
		}
		defer writer.Close()
		// Opening the FIFO proves the old state was loaded before input IO.
		// A second real CLI process grows that lock before the first resumes.
		lockGrow(t, input)
		result(t, invoke(t, base, args...), 0)
		competitor := snapshot(t, root)
		if _, err := writer.Write(original); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		err := cmd.Wait()
		exit := 0
		if err != nil {
			if x, ok := err.(*exec.ExitError); ok {
				exit = x.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if ctx.Err() != nil {
			t.Fatal("stale process timed out", ctx.Err())
		}
		fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_NAME_LOCK_STALE")
		if !reflect.DeepEqual(competitor, snapshot(t, root)) {
			t.Fatal("stale nested native default write/check replaced competitor bytes/modes/mtimes")
		}
	}
}
