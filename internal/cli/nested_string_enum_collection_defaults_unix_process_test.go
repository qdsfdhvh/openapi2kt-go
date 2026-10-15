//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestNestedStringEnumCollectionDefaultsNativeCASAndReadonly(t *testing.T) {
	for _, check := range []bool{false, true} {
		base := t.TempDir()
		input := nestedStringEnumLockInput(t, base)
		root := filepath.Join(base, "out")
		args := lockArgs(input, root)
		result(t, invoke(t, base, args...), 0)
		original := read(t, input)
		before := snapshot(t, root)
		if err := os.Chmod(root, 0500); err != nil {
			t.Fatal(err)
		}
		result(t, invoke(t, base, append(args, "--check")...), 0)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("readonly check/validate changed state")
		}
		if err := os.Chmod(root, 0700); err != nil {
			t.Fatal(err)
		}
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
		var doc, paths, alpha, get map[string]json.RawMessage
		if err := json.Unmarshal(original, &doc); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(doc["paths"], &paths); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(paths["/alpha"], &alpha); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(alpha["get"], &get); err != nil {
			t.Fatal(err)
		}
		get["operationId"] = json.RawMessage(`"aardvark"`)
		alpha["get"], _ = json.Marshal(get)
		paths["/aardvark"], _ = json.Marshal(alpha)
		doc["paths"], _ = json.Marshal(paths)
		grown, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		put(t, input, grown)
		result(t, invoke(t, base, args...), 0)
		competitor := snapshot(t, root)
		if _, err := writer.Write(original); err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
		err = cmd.Wait()
		exit := 0
		if err != nil {
			if x, ok := err.(*exec.ExitError); ok {
				exit = x.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_NAME_LOCK_STALE")
		if !reflect.DeepEqual(competitor, snapshot(t, root)) {
			t.Fatal("stale nested String enum default write/check replaced competitor")
		}
	}
}
