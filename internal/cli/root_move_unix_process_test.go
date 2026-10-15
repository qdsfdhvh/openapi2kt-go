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

func TestRootMoveNativeUnixSourceAndTargetBusyReadonly(t *testing.T) {
	for _, which := range []string{"source", "target"} {
		p := newProject(t)
		result(t, p.generate(t), 0)
		dst := filepath.Join(p.Root, "destination")
		if e := os.Mkdir(dst, 0700); e != nil {
			t.Fatal(e)
		}
		lockRoot := p.Output
		if which == "target" {
			lockRoot = dst
		}
		lock, e := os.OpenFile(filepath.Join(lockRoot, ".openapi-kt.lock"), os.O_CREATE|os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
			t.Fatal(e)
		}
		args := []string{"generate", "--input", p.Input, "--output", dst, "--move-output-from", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json"}
		a, b := snapshot(t, p.Output), snapshot(t, dst)
		result(t, invoke(t, p.Root, append(args, "--check")...), 3)
		result(t, invoke(t, p.Root, lifecycleValidate(args)...), 0)
		fails(t, invoke(t, p.Root, args...), 4, "E_OUTPUT_BUSY")
		// Acquisition can create a missing coordination sidecar; it must preserve
		// every previously existing payload and metadata on both sides.
		for root, before := range map[string]map[string]snap{p.Output: a, dst: b} {
			after := snapshot(t, root)
			for path, old := range before {
				if !reflect.DeepEqual(old, after[path]) {
					t.Fatal("busy changed old output")
				}
			}
		}
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		result(t, invoke(t, p.Root, args...), 0)
	}
}

func TestRootMoveNativeUnixFIFOExactManifestAndPeerStateCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, mutation := range []string{"manifest", "peer-state", "code"} {
			t.Run(mode+"/"+mutation, func(t *testing.T) {
				base := t.TempDir()
				names := []string{"alpha", "beta"}
				lifecycleConfigInputs(t, base, names)
				cfg := lifecycleConfig(t, base, names, true, false, true)
				args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
				result(t, invoke(t, base, args...), 0)
				src, dst := filepath.Join(base, "generated"), filepath.Join(base, "new")
				original := read(t, filepath.Join(base, "alpha.json"))
				fifo := filepath.Join(base, "alpha.json")
				if e := os.Remove(fifo); e != nil {
					t.Fatal(e)
				}
				if e := syscall.Mkfifo(fifo, 0600); e != nil {
					t.Fatal(e)
				}
				run := append(append([]string(nil), args...), "--output", dst, "--move-output-from", src)
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
				code := "E_OUTPUT_ROOT_MOVE_STALE"
				switch mutation {
				case "manifest":
					path := filepath.Join(src, ".openapi-kt-manifest.json")
					put(t, path, append(read(t, path), ' '))
				case "peer-state":
					path := filepath.Join(src, namedLockPath("beta"))
					put(t, path, append(read(t, path), ' '))
					code = "E_NAME_LOCK_MODIFIED"
				case "code":
					before := snapshot(t, src)
					for path := range before {
						if filepath.Ext(path) == ".kt" {
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
				if _, e := os.Stat(dst); !os.IsNotExist(e) {
					t.Fatal("CAS rejection created target")
				}
			})
		}
	}
}
