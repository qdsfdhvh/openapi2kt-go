//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExistingLockedAdoptionUnixBusyReadonlyAndCompleteLocks(t *testing.T) {
	updates := map[string]ExistingLockedAdoptionUpdate{}
	for i := 0; i < 2; i++ {
		root, u := existingLockedFixture(t)
		updates[root] = u
	}
	roots := sortedKeys(updates)
	f, err := os.OpenFile(filepath.Join(roots[1], lockName), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := tryLock(f); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, roots[0])
	if changed, err := WriteExistingLockedAdoption(roots[1], updates[roots[1]], true); err != nil || !changed {
		t.Fatal(changed, err)
	}
	_, e := writeOwnersBatchWith(updates, false, nil, makeExistingLockedAdoptionPlan)
	errorCode(t, e, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, roots[0])) {
		t.Fatal("busy staged earlier root")
	}
	unlock(f)
	prepared := []int{}
	_, e = writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, _ string) error {
		if phase == "prepare" {
			prepared = append(prepared, ri)
		}
		if phase == "apply" && op == 0 {
			for _, root := range roots {
				lock, err := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
				if err != nil {
					t.Fatal(err)
				}
				err = tryLock(lock)
				if err == nil {
					unlock(lock)
				}
				lock.Close()
				if err == nil {
					t.Fatal("missing complete lock set")
				}
			}
		}
		return nil
	}, makeExistingLockedAdoptionPlan)
	if e != nil || !reflect.DeepEqual(prepared, []int{0, 1}) {
		t.Fatal(e, prepared)
	}
}

func TestExistingLockedAdoptionUnixBindingAndSymlinkGuards(t *testing.T) {
	for _, kind := range []string{"manifest", "donor", "peer", "destination", "kotlin", "root"} {
		t.Run(kind, func(t *testing.T) {
			root, u := existingLockedFixture(t)
			path := root
			switch kind {
			case "manifest":
				path = filepath.Join(root, manifestName)
			case "donor":
				path = filepath.Join(root, ownerNameLockFile("left"))
			case "peer":
				path = filepath.Join(root, ownerNameLockFile("right"))
			case "destination":
				path = filepath.Join(root, ownerNameLockFile("consumer"))
			case "kotlin":
				path = filepath.Join(root, u.Previous.Previous.Files[0].Path)
			}
			target := path + "-old"
			if kind != "destination" {
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
			} else {
				put(t, target, []byte("synthetic unowned lock"))
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			for _, check := range []bool{true, false} {
				_, e := WriteExistingLockedAdoption(root, u, check)
				errorCode(t, e, "E_OUTPUT_PATH")
			}
		})
	}
	root, u := existingLockedFixture(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(filepath.Dir(root), alias); err != nil {
		t.Fatal(err)
	}
	via := filepath.Join(alias, filepath.Base(root))
	s, e := ReadExistingLockedAdoptionState(via, "consumer", "left")
	if e != nil {
		t.Fatal(e)
	}
	u.Previous = s
	other, _ := existingLockedFixture(t)
	holder := t.TempDir()
	if err := os.Rename(other, filepath.Join(holder, filepath.Base(root))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(holder, alias); err != nil {
		t.Fatal(err)
	}
	for _, check := range []bool{true, false} {
		_, e := WriteExistingLockedAdoption(via, u, check)
		errorCode(t, e, "E_LOCKED_OWNER_ADOPTION_STALE")
	}
}
