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

func TestLockedOwnerReassignmentNativeUnixFIFOPreInputWholeStateCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, mutation := range []string{"manifest", "peer-state", "code", "attributes"} {
			t.Run(mode+"/"+mutation, func(t *testing.T) {
				base, cfg, src := reassignmentNativeFixture(t, "data_class", false, false)
				args := []string{"generate", "--config", cfg, "--reassign-locked-owners", "--diagnostic-format=json"}
				original := read(t, filepath.Join(base, "api.json"))
				fifo := filepath.Join(base, "api.json")
				if e := os.Remove(fifo); e != nil {
					t.Fatal(e)
				}
				if e := syscall.Mkfifo(fifo, 0600); e != nil {
					t.Fatal(e)
				}
				run := append([]string(nil), args...)
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
					t.Fatal("FIFO state-read barrier timed out")
				}
				defer writer.Close()
				code := "E_LOCKED_OWNER_REASSIGN_STALE"
				switch mutation {
				case "manifest":
					path := filepath.Join(src, ".openapi-kt-manifest.json")
					put(t, path, append(read(t, path), ' '))
				case "peer-state":
					path := filepath.Join(src, namedLockPath("api"))
					put(t, path, append(read(t, path), ' '))
					code = "E_NAME_LOCK_MODIFIED"
				case "attributes":
					for path := range snapshot(t, src) {
						if filepath.Ext(path) == ".kt" {
							full := filepath.Join(src, path)
							stamp := time.Unix(1400000000, 0)
							if e := os.Chtimes(full, stamp, stamp); e != nil {
								t.Fatal(e)
							}
							break
						}
					}
				case "code":
					before := snapshot(t, src)
					for path := range before {
						if filepath.Ext(path) == ".kt" && strings.Contains(path, "split") {
							put(t, filepath.Join(src, path), []byte("edited concurrent code"))
							break
						}
					}
					code = "E_OUTPUT_MODIFIED"
				}
				before := snapshot(t, src)
				if _, e := writer.Write(original); e != nil {
					t.Fatal(e)
				}
				if e := writer.Close(); e != nil {
					t.Fatal(e)
				}
				e := cmd.Wait()
				exit := 0
				if e != nil {
					if status, ok := e.(*exec.ExitError); ok {
						exit = status.ExitCode()
					} else {
						t.Fatal(e)
					}
				}
				fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, code)
				if !reflect.DeepEqual(before, snapshot(t, src)) {
					t.Fatal("CAS destroyed intervening source")
				}

			})
		}
	}
}
