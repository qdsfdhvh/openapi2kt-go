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

func TestNameLockLifecycleNativeUnixBusyWriteReadonlyCheckValidate(t *testing.T) {
	for _, configured := range []bool{false, true} {
		for _, action := range []string{"reset", "disable"} {
			base := t.TempDir()
			input := lockInput(t, base)
			root := filepath.Join(base, "generated")
			args := lifecycleArgs(input, root, true)
			if configured {
				lifecycleConfigInputs(t, base, []string{"alpha", "beta"})
				cfg := lifecycleConfig(t, base, []string{"alpha", "beta"}, true, true, false)
				args = []string{"generate", "--config", cfg, "--diagnostic-format=json"}
				root = filepath.Join(base, "beta-output")
			}
			result(t, invoke(t, base, args...), 0)
			if configured {
				for _, name := range []string{"alpha", "beta"} {
					lockGrow(t, filepath.Join(base, name+".json"))
				}
				if action == "disable" {
					lifecycleConfig(t, base, []string{"alpha", "beta"}, false, true, false)
				}
			} else {
				lockGrow(t, input)
				args = lifecycleArgs(input, root, action == "reset")
			}
			f, err := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
				_ = f.Close()
				t.Fatal(err)
			}
			before := snapshot(t, base)
			actionArgs := lifecycleAction(args, action)
			checked := lifecycleMigrations(t, invoke(t, base, append(actionArgs, "--check")...), 3)
			validated := lifecycleMigrations(t, invoke(t, base, lifecycleValidate(actionArgs)...), 0)
			if !reflect.DeepEqual(checked, validated) {
				t.Fatal("busy root altered read-only lifecycle report")
			}
			fails(t, invoke(t, base, actionArgs...), 4, "E_OUTPUT_BUSY")
			if !reflect.DeepEqual(before, snapshot(t, base)) {
				t.Fatal("busy lifecycle write/check/validate mutated output")
			}
			if configured {
				// The later busy root must release the earlier acquired root lock.
				lifecycleMigrations(t, invoke(t, base, append(actionArgs, "--spec=alpha")...), 0)
			}
			if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestNameLockLifecycleNativeUnixStaleStateCASWithFIFO(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		for _, check := range []bool{false, true} {
			base := t.TempDir()
			input := lockInput(t, base)
			root := filepath.Join(base, "generated")
			args := lifecycleArgs(input, root, true)
			result(t, invoke(t, base, args...), 0)
			original := read(t, input)
			fifo := filepath.Join(base, "lifecycle-barrier.json")
			if err := syscall.Mkfifo(fifo, 0600); err != nil {
				t.Fatal(err)
			}
			blocked := lifecycleAction(lifecycleArgs(fifo, root, action == "reset"), action)
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
				t.Fatal("lifecycle FIFO state-read barrier timed out")
			}
			defer writer.Close()
			// The blocked lifecycle read old state before opening its input.
			// A cooperating writer now persists a new occurrence and state hash.
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
			code := 0
			if err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					code = exit.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			out := invocation{Code: code, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
			fails(t, out, 4, "E_NAME_LOCK_STALE")
			lifecycleNoMigrations(t, out)
			if !reflect.DeepEqual(competitor, snapshot(t, root)) {
				t.Fatal("lifecycle stale CAS replaced competitor state/source")
			}
		}
	}
}
