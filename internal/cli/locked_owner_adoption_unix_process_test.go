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

func TestLockedOwnerAdoptionNativeUnixBusyReadonly(t *testing.T) {
	base, cfg, root, _ := lockedAdoptionFixture(t)
	f, e := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	before := snapshot(t, root)
	run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(run, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("busy/read-only changed output")
	}
}

func TestLockedOwnerAdoptionNativeUnixFIFOExactSourceCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, changed := range []string{"manifest", "kotlin", "validpeer", "alias"} {
			t.Run(mode+"/"+changed, func(t *testing.T) {
				base, cfg, root, direct := lockedAdoptionFixture(t)
				original := read(t, filepath.Join(base, "catalog.json"))
				fifo := filepath.Join(base, "barrier.json")
				if e := syscall.Mkfifo(fifo, 0600); e != nil {
					t.Fatal(e)
				}
				put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "catalog.json", "barrier.json")))
				alias := filepath.Join(base, "ancestor")
				if changed == "alias" {
					if e := os.Symlink(base, alias); e != nil {
						t.Fatal(e)
					}
					put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "output_dir='generated'", "output_dir='ancestor/generated'")))
				}
				run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
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
					t.Fatal("pre-input state barrier timed out")
				}
				defer writer.Close()
				code := "E_LOCKED_OWNER_ADOPTION_STALE"
				switch changed {
				case "manifest":
					path := filepath.Join(root, ".openapi-kt-manifest.json")
					put(t, path, append(read(t, path), ' '))
				case "kotlin":
					for path := range snapshot(t, root) {
						if strings.HasSuffix(path, ".kt") {
							put(t, filepath.Join(root, path), []byte("edited after state read"))
							break
						}
					}
					code = "E_OUTPUT_MODIFIED"
				case "validpeer":
					lockGrow(t, filepath.Join(base, "catalog.json"))
					result(t, invoke(t, base, direct...), 0)
				case "alias":
					holder := filepath.Join(base, "other")
					if e := os.Mkdir(holder, 0700); e != nil {
						t.Fatal(e)
					}
					control := append([]string(nil), direct...)
					for i, arg := range control {
						if arg == root {
							control[i] = filepath.Join(holder, "generated")
						}
					}
					result(t, invoke(t, base, control...), 0)
					if e := os.Remove(alias); e != nil {
						t.Fatal(e)
					}
					if e := os.Symlink(holder, alias); e != nil {
						t.Fatal(e)
					}
				}
				before := snapshot(t, root)
				if _, e := writer.Write(original); e != nil {
					t.Fatal(e)
				}
				if e := writer.Close(); e != nil {
					t.Fatal(e)
				}
				e := cmd.Wait()
				exit := 0
				if e != nil {
					if x, ok := e.(*exec.ExitError); ok {
						exit = x.ExitCode()
					} else {
						t.Fatal(e)
					}
				}
				fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("stale candidate changed current output")
				}
			})
		}
	}
}
