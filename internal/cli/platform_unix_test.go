//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestActualGoProcessCooperativeLockAndReadonlyCheck(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	lock, err := os.OpenFile(filepath.Join(p.Output, ".openapi-kt.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	fails(t, p.generate(t), 4, "E_OUTPUT_BUSY")
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Payload"] = map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}
	})
	result(t, p.generate(t, "--check"), 3)
	fails(t, p.generate(t), 4, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(snapshot(t, p.Output), before) {
		t.Fatal("busy/check mutated existing output")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	result(t, p.generate(t), 0)
}

func TestActualGoProcessSymlinkRootsParentsAndDestinations(t *testing.T) {
	for _, where := range []string{"root", "parent", "destination", "manifest", "lock"} {
		t.Run(where, func(t *testing.T) {
			p := newProject(t)
			outside := filepath.Join(p.Root, "outside")
			if err := os.Mkdir(outside, 0700); err != nil {
				t.Fatal(err)
			}
			manual := filepath.Join(outside, "manual.kt")
			put(t, manual, []byte("keep"))
			link := func(dest string) {
				t.Helper()
				if err := os.Symlink(manual, dest); err != nil {
					t.Fatal(err)
				}
			}
			if where == "root" {
				if err := os.Symlink(outside, p.Output); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Mkdir(p.Output, 0700); err != nil {
					t.Fatal(err)
				}
				switch where {
				case "parent":
					if err := os.Symlink(outside, filepath.Join(p.Output, "org")); err != nil {
						t.Fatal(err)
					}
				case "destination":
					path := filepath.Join(p.Output, "org/example/catalog/models")
					if err := os.MkdirAll(path, 0700); err != nil {
						t.Fatal(err)
					}
					link(filepath.Join(path, "Item.kt"))
				case "manifest":
					link(filepath.Join(p.Output, ".openapi-kt-manifest.json"))
				case "lock":
					link(filepath.Join(p.Output, ".openapi-kt.lock"))
				}
			}
			for _, check := range []bool{true, false} {
				if where == "lock" && check {
					continue
				}
				args := []string{}
				if check {
					args = append(args, "--check")
				}
				fails(t, p.generate(t, args...), 4, "E_OUTPUT_PATH")
			}
			if string(read(t, manual)) != "keep" {
				t.Fatal("outside file modified")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 1 {
				t.Fatal("escaped root", entries, err)
			}
		})
	}
}
