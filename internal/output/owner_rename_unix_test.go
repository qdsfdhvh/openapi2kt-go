//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOwnerRenameCompleteSortedLocksAndBusyRelease(t *testing.T) {
	updates := map[string]OwnerRenameUpdate{}
	for i := 0; i < 2; i++ {
		root, _, u := renameFixture(t, true)
		updates[root] = u
	}
	roots := sortedKeys(updates)
	before := map[string]map[string]snapshotEntry{}
	for _, root := range roots {
		before[root] = snapshot(t, root)
	}
	busy, e := os.OpenFile(filepath.Join(roots[1], lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer busy.Close()
	if e := tryLock(busy); e != nil {
		t.Fatal(e)
	}
	changed, err := WriteOwnerRenamesBatch(updates, true)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	_, err = WriteOwnerRenamesBatch(updates, false)
	errorCode(t, err, "E_OUTPUT_BUSY")
	for _, root := range roots {
		if !reflect.DeepEqual(before[root], snapshot(t, root)) {
			t.Fatal("busy batch changed output")
		}
	}
	first, e := os.OpenFile(filepath.Join(roots[0], lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	if e := tryLock(first); e != nil {
		t.Fatal("first lock leaked", e)
	}
	unlock(first)
	_ = first.Close()
	unlock(busy)
	seen := []int{}
	_, err = writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, stage string) error {
		if phase == "prepare" {
			seen = append(seen, ri)
		}
		if phase == "apply" && op == 0 {
			for _, root := range roots {
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
					t.Fatal("rename missing complete root lock set")
				}
			}
			canonical, e := canonicalRoot(roots[ri])
			if e != nil || filepath.Dir(stage) != canonical {
				t.Fatal("root order", e)
			}
		}
		return nil
	}, makeOwnerRenamePlan)
	if err != nil || !reflect.DeepEqual(seen, []int{0, 1}) {
		t.Fatal(err, seen)
	}
}

func TestOwnerRenameFilesystemSymlinksAliasesAndStateCAS(t *testing.T) {
	for _, entry := range []string{"kotlin", "state", "destination", "parent", "root"} {
		root, a, u := renameFixture(t, true)
		path := root
		switch entry {
		case "kotlin":
			path = filepath.Join(root, a.Files[0].Path)
		case "state":
			path = filepath.Join(root, ownerNameLockFile("old"))
		case "destination":
			path = filepath.Join(root, ownerNameLockFile("new"))
		case "parent":
			path = filepath.Dir(filepath.Join(root, a.Files[0].Path))
		}
		if entry != "destination" {
			backup := path + "-original"
			if e := os.Rename(path, backup); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(backup, path); e != nil {
				t.Fatal(e)
			}
		} else if e := os.Symlink(filepath.Join(root, ownerNameLockFile("old")), path); e != nil {
			t.Fatal(e)
		}
		for _, check := range []bool{true, false} {
			_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check)
			errorCode(t, e, "E_OUTPUT_PATH")
		}
	}
	root, _, u := renameFixture(t, true)
	alias := filepath.Join(t.TempDir(), "alias")
	if e := os.Symlink(filepath.Dir(root), alias); e != nil {
		t.Fatal(e)
	}
	_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u, filepath.Join(alias, filepath.Base(root)): u}, false)
	errorCode(t, e, "E_OUTPUT_ROOTS")
	// A valid noncanonical serialization with identical semantic lock allocations
	// and matching manifest bytes still has to satisfy the payload-byte CAS.
	stateFile := filepath.Join(root, ownerNameLockFile("peer"))
	prior := mustRead(t, stateFile)
	u.Previous.NameLocks["peer"] = append(append([]byte(nil), prior...), ' ')
	before := snapshot(t, root)
	_, e = WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false)
	errorCode(t, e, "E_OWNER_RENAME_STALE")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("payload CAS changed output")
	}
}
