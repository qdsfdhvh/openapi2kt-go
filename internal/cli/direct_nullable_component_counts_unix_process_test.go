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

func TestDirectNullableComponentCountsNativeReadonlyAndStaleCAS(t *testing.T) {
	for _, check := range []bool{false, true} {
		base := t.TempDir()
		input := directNullableCountLockInput(t, base)
		output := filepath.Join(base, "out")
		args := lockArgs(input, output)
		result(t, invoke(t, base, args...), 0)
		original := read(t, input)
		before := snapshot(t, output)
		if err := os.Chmod(output, 0500); err != nil {
			t.Fatal(err)
		}
		result(t, invoke(t, base, append(args, "--check")...), 0)
		validate := append([]string(nil), args...)
		validate[0] = "validate"
		result(t, invoke(t, base, validate...), 0)
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("readonly native changed count/lock state")
		}
		if err := os.Chmod(output, 0700); err != nil {
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
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		opened := make(chan *os.File, 1)
		failures := make(chan error, 1)
		go func() {
			f, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
			if err != nil {
				failures <- err
			} else {
				opened <- f
			}
		}()
		var writer *os.File
		select {
		case writer = <-opened:
		case err := <-failures:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal("FIFO state-read barrier timeout")
		}
		var competitorDoc map[string]any
		if err := json.Unmarshal(original, &competitorDoc); err != nil {
			t.Fatal(err)
		}
		paths := competitorDoc["paths"].(map[string]any)
		newOperation := map[string]any{}
		operationBytes, err := json.Marshal(paths["/alpha"])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(operationBytes, &newOperation); err != nil {
			t.Fatal(err)
		}
		newOperation["get"].(map[string]any)["operationId"] = "aardvark"
		paths["/aardvark"] = newOperation
		competitorInput, err := json.Marshal(competitorDoc)
		if err != nil {
			t.Fatal(err)
		}
		put(t, input, competitorInput)
		result(t, invoke(t, base, args...), 0)
		competitor := snapshot(t, output)
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
		if !reflect.DeepEqual(competitor, snapshot(t, output)) {
			t.Fatal("stale nullable count writer/check replaced competitor state")
		}
	}
}
