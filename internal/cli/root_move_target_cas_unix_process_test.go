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

func TestRootMoveNativeFIFOInitialTargetAndDeclaredBindingCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, binding := range []string{"target", "declared"} {
			t.Run(mode+"/"+binding, func(t *testing.T) {
				base := t.TempDir()
				names := []string{"alpha"}
				lifecycleConfigInputs(t, base, names)
				cfg := lifecycleConfig(t, base, names, true, false, false)
				args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
				result(t, invoke(t, base, args...), 0)
				src := filepath.Join(base, "generated")
				a, b := filepath.Join(base, "A"), filepath.Join(base, "B")
				for _, root := range []string{a, b} {
					if e := os.Mkdir(root, 0700); e != nil {
						t.Fatal(e)
					}
				}
				alias := filepath.Join(base, "alias")
				if e := os.Symlink(a, alias); e != nil {
					t.Fatal(e)
				}
				declared, target := "A/new", filepath.Join(alias, "new")
				if binding == "declared" {
					declared = "alias/new"
					target = filepath.Join(a, "new")
				}
				put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "output_dir='generated'", "output_dir='"+declared+"'")))
				input := filepath.Join(base, "alpha.json")
				original := read(t, input)
				if e := os.Remove(input); e != nil {
					t.Fatal(e)
				}
				if e := syscall.Mkfifo(input, 0600); e != nil {
					t.Fatal(e)
				}
				run := append(append([]string(nil), args...), "--move-output-from", src, "--output", target)
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
					f, e := os.OpenFile(input, os.O_WRONLY, 0600)
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
					t.Fatal("binding FIFO barrier timed out")
				}
				defer writer.Close()
				if e := os.Remove(alias); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(b, alias); e != nil {
					t.Fatal(e)
				}
				before := snapshot(t, src)
				if _, e := writer.Write(original); e != nil {
					t.Fatal(e)
				}
				if e := writer.Close(); e != nil {
					t.Fatal(e)
				}
				err := cmd.Wait()
				code := 0
				if err != nil {
					if status, ok := err.(*exec.ExitError); ok {
						code = status.ExitCode()
					} else {
						t.Fatal(err)
					}
				}
				fails(t, invocation{Code: code, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_OUTPUT_ROOT_MOVE_STALE")
				if !reflect.DeepEqual(before, snapshot(t, src)) {
					t.Fatal("binding change removed source")
				}
				for _, root := range []string{a, b} {
					if _, e := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(e) {
						t.Fatal("binding change installed output", root)
					}
				}
			})
		}
	}
}
