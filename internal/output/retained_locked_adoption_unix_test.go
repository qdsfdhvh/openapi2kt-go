//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRetainedLockedAdoptionUnixBusyAndLockedCAS(t *testing.T) {
	root, u := retainedAdoptionFixture(t, []string{"part"})
	before := snapshot(t, root)
	f, err := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err = tryLock(f); err != nil {
		t.Fatal(err)
	}
	if changed, e := WriteRetainedLockedAdoption(root, u, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	_, e := WriteRetainedLockedAdoption(root, u, false)
	errorCode(t, e, "E_OUTPUT_BUSY")
	unlock(f)
	_ = f.Close()
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("busy/readonly reassignment wrote")
	}
	_, e = writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, _ string) error {
		if phase == "apply" && op == 0 {
			f, err := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			err = tryLock(f)
			if err == nil {
				unlock(f)
			}
			_ = f.Close()
			if err == nil {
				t.Fatal("reassignment apply lacks cooperative lock")
			}
		}
		return nil
	}, makeRetainedLockedAdoptionPlan)
	if e != nil {
		t.Fatal(e)
	}
}

func TestRetainedLockedAdoptionUnixBindingAndPhysicalRootCAS(t *testing.T) {
	for _, kind := range []string{"binding", "identity"} {
		root, u := retainedAdoptionFixture(t, []string{"part"})
		before := snapshot(t, root)
		raw := root
		alias := ""
		alternate := t.TempDir()
		if kind == "binding" {
			alias = filepath.Join(t.TempDir(), "alias")
			if err := os.Symlink(filepath.Dir(root), alias); err != nil {
				t.Fatal(err)
			}
			raw = filepath.Join(alias, filepath.Base(root))
			s, e := ReadRetainedLockedAdoptionState(raw, "original", u.Previous.OwnerNames)
			if e != nil {
				t.Fatal(e)
			}
			u.Previous = s
		}
		_, e := writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{raw: u}, false, func(_ int, phase string, _ int, _ string) error {
			if phase != "prelock" {
				return nil
			}
			if kind == "binding" {
				if e := os.Remove(alias); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(alternate, alias); e != nil {
					t.Fatal(e)
				}
			} else {
				backup := filepath.Join(t.TempDir(), "original")
				if e := os.Rename(root, backup); e != nil {
					t.Fatal(e)
				}
				if e := os.MkdirAll(root, 0700); e != nil {
					t.Fatal(e)
				}
				for path, entry := range before {
					if entry.Directory {
						if e := os.MkdirAll(filepath.Join(root, path), 0700); e != nil {
							t.Fatal(e)
						}
						continue
					}
					full := filepath.Join(root, path)
					if e := os.MkdirAll(filepath.Dir(full), 0700); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(full, []byte(entry.Data), entry.Mode); e != nil {
						t.Fatal(e)
					}
					if e := os.Chtimes(full, entry.Modified, entry.Modified); e != nil {
						t.Fatal(e)
					}
				}
				before = snapshot(t, root)
			}
			return nil
		}, makeRetainedLockedAdoptionPlan)
		errorCode(t, e, "E_RETAINED_LOCKED_ADOPTION_STALE")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("root rebinding/replacement altered observed output", kind)
		}
	}
}

func TestRetainedLockedAdoptionUnixObstructedRollbackKeepsEvidence(t *testing.T) {
	root, u := retainedAdoptionFixture(t, []string{"part"})
	path := nameLockFile
	old := u.Previous.NameLock
	_, e := writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, stage string) error {
		if phase == "apply" {
			var j journal
			if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
				t.Fatal(e)
			}
			if j.Operations[op].Path == manifestName {
				obstructLifecycleRollback(t, root, path)
				return errors.New("synthetic reassignment rollback obstruction")
			}
		}
		return nil
	}, makeRetainedLockedAdoptionPlan)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, root, path, old)
	before := snapshot(t, root)
	_, e = WriteRetainedLockedAdoption(root, u, true)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("recovery evidence altered")
	}
}
