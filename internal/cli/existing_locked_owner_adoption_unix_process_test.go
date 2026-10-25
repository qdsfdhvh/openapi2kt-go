//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestExistingV4OwnerAdoptionNativeUnixBusyReadonly(t *testing.T) {
	base, cfg, root, _ := existingDonorFixture(t)
	f, err := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	run := []string{"generate", "--config", cfg, "--spec=api", "--adopt-locked-owner", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"}
	before := snapshot(t, root)
	result(t, invoke(t, base, append(append([]string{}, run...), "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("busy/readonly changed state")
	}
}

func TestExistingV4OwnerAdoptionNativeUnixFIFOCompleteSourceCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, changed := range []string{"manifest", "kotlin", "validpeer", "alias", "destination"} {
			t.Run(mode+"/"+changed, func(t *testing.T) {
				base, cfg, root, _ := existingDonorFixture(t)
				original := read(t, filepath.Join(base, "api.json"))
				fifo := filepath.Join(base, "barrier.json")
				if err := syscall.Mkfifo(fifo, 0600); err != nil {
					t.Fatal(err)
				}
				text := strings.Replace(string(read(t, cfg)), "input='api.json'", "input='barrier.json'", 1)
				alias := filepath.Join(base, "ancestor")
				if changed == "alias" {
					if err := os.Symlink(base, alias); err != nil {
						t.Fatal(err)
					}
					text = strings.Replace(text, "output_dir='generated'", "output_dir='ancestor/generated'", 1)
				}
				put(t, cfg, []byte(text))
				run := []string{"generate", "--config", cfg, "--spec=api", "--adopt-locked-owner", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"}
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
					t.Fatal("existing-owner pre-input source barrier timed out")
				}
				defer writer.Close()
				code := "E_LOCKED_OWNER_ADOPTION_STALE"
				switch changed {
				case "manifest":
					path := filepath.Join(root, ".openapi-kt-manifest.json")
					put(t, path, append(read(t, path), '\n'))
				case "kotlin":
					for path := range snapshot(t, root) {
						if strings.HasSuffix(path, ".kt") {
							put(t, filepath.Join(root, path), []byte("// intervening edit"))
							break
						}
					}
					code = "E_OUTPUT_MODIFIED"
				case "destination":
					put(t, filepath.Join(root, namedLockPath("api")), []byte("foreign"))
					code = "E_OUTPUT_COLLISION"
				case "validpeer":
					path := filepath.Join(root, namedLockPath("catalog"))
					payload := append(read(t, path), ' ')
					put(t, path, payload)
					var m map[string]json.RawMessage
					var hashes map[string]string
					manifestPath := filepath.Join(root, ".openapi-kt-manifest.json")
					if err := json.Unmarshal(read(t, manifestPath), &m); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(m["name_locks"], &hashes); err != nil {
						t.Fatal(err)
					}
					hashes["catalog"] = fmt.Sprintf("%x", sha256.Sum256(payload))
					m["name_locks"], _ = json.Marshal(hashes)
					data, _ := json.Marshal(m)
					put(t, manifestPath, data)
				case "alias":
					holder := filepath.Join(base, "other")
					copyCfg := filepath.Join(base, "copy.toml")
					copyText := strings.Replace(string(read(t, cfg)), "output_dir='ancestor/generated'", "output_dir='other/generated'", 1)
					copyText = strings.Replace(copyText, "input='barrier.json'", "input='api.json'", 1)
					put(t, copyCfg, []byte(copyText))
					result(t, invoke(t, base, "generate", "--config", copyCfg, "--diagnostic-format=json"), 0)
					if err := os.Remove(alias); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(holder, alias); err != nil {
						t.Fatal(err)
					}
				}
				before := snapshot(t, root)
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
				fails(t, invocation{Code: exit, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, 4, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("stale existing-owner adoption overwrote intervening source")
				}
			})
		}
	}
}
