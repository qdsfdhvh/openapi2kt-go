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

func TestRootMoveDisjointNativeUnrelatedAliasFIFO(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		base := t.TempDir()
		peerBase, otherBase := t.TempDir(), t.TempDir()
		alias := filepath.Join(base, "peer-alias")
		if err := os.Symlink(peerBase, alias); err != nil {
			t.Fatal(err)
		}
		names := []string{"alpha", "beta", "gamma"}
		lifecycleConfigInputs(t, base, names)
		cfg := lifecycleConfig(t, base, names, true, false, false)
		text := strings.Replace(string(read(t, cfg)), "name=\"gamma\"", "name=\"gamma\"\noutput_dir='peer-alias/output'", 1)
		put(t, cfg, []byte(text))
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		src, dst := filepath.Join(base, "generated"), filepath.Join(base, "new")
		before, peerBefore := snapshot(t, src), snapshot(t, filepath.Join(peerBase, "output"))
		original := read(t, filepath.Join(base, "alpha.json"))
		fifo := filepath.Join(base, "alpha.json")
		if err := os.Remove(fifo); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(fifo, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(base, "gamma.json")); err != nil {
			t.Fatal(err)
		}
		run := append(append([]string(nil), args...), "--move-output-from", src, "--output", dst, "--spec=alpha")
		if mode == "check" {
			run = append(run, "--check")
		}
		if mode == "validate" {
			run = lifecycleValidate(run)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, run...)
		cmd.Dir = base
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer cmd.Process.Kill()
		opened := make(chan *os.File, 1)
		openErrors := make(chan error, 1)
		go func() {
			writer, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
			if err != nil {
				openErrors <- err
			} else {
				opened <- writer
			}
		}()
		var writer *os.File
		select {
		case writer = <-opened:
		case err := <-openErrors:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal("FIFO source binding barrier timed out")
		}
		defer writer.Close()
		if err := os.Remove(alias); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(otherBase, alias); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(original); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		err := cmd.Wait()
		exit := 0
		if err != nil {
			if status, ok := err.(*exec.ExitError); ok {
				exit = status.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, "E_OUTPUT_ROOT_MOVE_STALE")
		if !reflect.DeepEqual(before, snapshot(t, src)) || !reflect.DeepEqual(peerBefore, snapshot(t, filepath.Join(peerBase, "output"))) {
			t.Fatal("root alias rebind lost moved/untouched output")
		}
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			t.Fatal("stale check created target")
		}
	}
}
