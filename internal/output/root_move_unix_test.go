//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestRootMoveUnixBusyCompleteSortedLocksAndStages(t *testing.T) {
	for _, busySource := range []bool{false, true} {
		src, dst, s := moveFixture(t, 4)
		busyRoot := dst
		if busySource {
			busyRoot = src
		}
		f, e := os.OpenFile(filepath.Join(busyRoot, lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := tryLock(f); e != nil {
			t.Fatal(e)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		if changed, e := MoveOutputRoot(src, dst, s, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		_, err := MoveOutputRoot(src, dst, s, false)
		errorCode(t, err, "E_OUTPUT_BUSY")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("busy/check changed output")
		}
		unlock(f)
		_ = f.Close()
		prepared := []int{}
		_, err = moveOutputRoot(src, dst, s, false, func(index int, phase string, op int, stage string) error {
			if phase == "prepare" {
				prepared = append(prepared, index)
			}
			if phase == "apply" && op == 0 {
				if !reflect.DeepEqual(prepared, []int{0, 1}) {
					t.Fatal("apply before all preparation", prepared)
				}
				for _, root := range []string{src, dst} {
					f, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
					if e != nil {
						t.Fatal(e)
					}
					e = tryLock(f)
					if e == nil {
						unlock(f)
					}
					_ = f.Close()
					if e == nil {
						t.Fatal("complete locks missing")
					}
					entries, e := os.ReadDir(root)
					if e != nil {
						t.Fatal(e)
					}
					found := false
					for _, entry := range entries {
						if len(entry.Name()) > len(stagePrefix) && entry.Name()[:len(stagePrefix)] == stagePrefix {
							found = true
						}
					}
					if !found {
						t.Fatal("root not staged")
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal("released lock leaked", err)
		}
	}
}

func TestRootMoveUnixSymlinksAncestorsOverlapAliasesAndFIFOEntries(t *testing.T) {
	for _, kind := range []string{"source-root", "target-root", "source-parent", "target-parent", "source-code", "target-code", "source-lock", "target-lock", "metadata", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			src, dst, s := moveFixture(t, 3)
			path := src
			switch kind {
			case "target-root":
				path = dst
			case "source-parent":
				path = filepath.Dir(filepath.Join(src, s.Files[0].Path))
			case "target-parent":
				path = filepath.Join(dst, "org")
				if e := os.MkdirAll(path, 0700); e != nil {
					t.Fatal(e)
				}
			case "source-code", "fifo":
				path = filepath.Join(src, s.Files[0].Path)
			case "target-code":
				path = filepath.Join(dst, s.Files[0].Path)
				if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
					t.Fatal(e)
				}
				put(t, path, []byte("unowned"))
			case "source-lock":
				path = filepath.Join(src, lockName)
			case "target-lock":
				path = filepath.Join(dst, lockName)
			case "metadata":
				path = filepath.Join(src, nameLockFile)
			}
			backup := path + "-real"
			if e := os.Rename(path, backup); e != nil {
				t.Fatal(e)
			}
			if kind == "fifo" {
				if e := syscall.Mkfifo(path, 0600); e != nil {
					t.Fatal(e)
				}
			} else if e := os.Symlink(backup, path); e != nil {
				t.Fatal(e)
			}
			// A target payload symlink is an unowned collision; busy-sidecar checks
			// occur only on apply, consistent with the existing read-only contract.
			check := kind != "source-lock" && kind != "target-lock"
			_, e := MoveOutputRoot(src, dst, s, check)
			if e == nil {
				t.Fatal("unsafe entry accepted", kind)
			}
		})
	}
	src, _, s := moveFixture(t, 1)
	for _, dst := range []string{filepath.Join(src, "nested"), filepath.Dir(src), filepath.Join(filepath.Dir(src), "A-SOURCE")} {
		_, e := MoveOutputRoot(src, dst, s, true)
		errorCode(t, e, "E_OUTPUT_ROOTS")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if e := os.Symlink(filepath.Dir(src), alias); e != nil {
		t.Fatal(e)
	}
	_, e := MoveOutputRoot(src, filepath.Join(alias, filepath.Base(src)), s, true)
	errorCode(t, e, "E_OUTPUT_ROOTS")
}

func TestRootMoveFailedRollbackRetainsSourceBackupAndRestoresTarget(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	oldDst := snapshot(t, dst)
	path := s.Files[0].Path
	_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, op int, stage string) error {
		if phase == "apply" && index == 1 {
			var j journal
			if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
				t.Fatal(e)
			}
			if j.Operations[op].Path == manifestName {
				obstructLifecycleRollback(t, src, path)
				return errors.New("synthetic obstruction")
			}
		}
		return nil
	})
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, src, path, s.Entries[path].Data)
	if !reflect.DeepEqual(oldDst, snapshot(t, dst)) {
		t.Fatal("target restoration skipped")
	}
	oldSrc := snapshot(t, src)
	for _, check := range []bool{true, false} {
		_, e := MoveOutputRoot(src, dst, s, check)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(oldSrc, snapshot(t, src)) {
			t.Fatal("recovery evidence changed")
		}
	}
}
