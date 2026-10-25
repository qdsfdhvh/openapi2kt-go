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
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSplitLockedAdoptionNativeUnixBusyReadonly(t *testing.T) {
	base := t.TempDir()
	input, cfg := splitAdoptionInputs(t, base)
	root := filepath.Join(base, "generated")
	result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
	f, err := os.OpenFile(filepath.Join(root, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	before := snapshot(t, root)
	args := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(args)...), 0)
	fails(t, invoke(t, base, args...), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("split busy/readonly changed output")
	}
}

func TestSplitLockedAdoptionNativeUnixFIFOOriginalSourceCAS(t *testing.T) {
	for _, mode := range []string{"apply", "check", "validate"} {
		for _, changed := range []string{"manifest", "kotlin", "validpeer", "alias", "destination"} {
			t.Run(mode+"/"+changed, func(t *testing.T) {
				base := t.TempDir()
				input, cfg := splitAdoptionInputs(t, base)
				root := filepath.Join(base, "generated")
				direct := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"}
				result(t, invoke(t, base, direct...), 0)
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
				run := []string{"generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"}
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
					t.Fatal("split pre-input source barrier timed out")
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
					put(t, filepath.Join(root, namedLockPath("models")), []byte("foreign"))
					code = "E_OUTPUT_COLLISION"
				case "validpeer":
					var doc map[string]json.RawMessage
					var components map[string]json.RawMessage
					var schemas map[string]json.RawMessage
					_ = json.Unmarshal(read(t, input), &doc)
					_ = json.Unmarshal(doc["components"], &components)
					_ = json.Unmarshal(components["schemas"], &schemas)
					schemas["C"] = json.RawMessage(`{"properties":{"added":{"type":"string"}}}`)
					components["schemas"], _ = json.Marshal(schemas)
					doc["components"], _ = json.Marshal(components)
					data, _ := json.Marshal(doc)
					put(t, input, data)
					result(t, invoke(t, base, direct...), 0)
				case "alias":
					holder := filepath.Join(base, "other")
					control := append([]string(nil), direct...)
					for i, arg := range control {
						if arg == root {
							control[i] = filepath.Join(holder, "generated")
						}
					}
					result(t, invoke(t, base, control...), 0)
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
					t.Fatal("stale split overwrote intervening source")
				}
			})
		}
	}
}
