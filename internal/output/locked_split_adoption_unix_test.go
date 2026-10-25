//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitLockedAdoptionUnixSortedLocksBusyAndReplanCAS(t *testing.T) {
	updates := map[string]SplitLockedAdoptionUpdate{}
	for i := 0; i < 2; i++ {
		root, u := splitLockedFixture(t)
		updates[root] = u
	}
	roots := sortedKeys(updates)
	f, e := os.OpenFile(filepath.Join(roots[1], lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e := tryLock(f); e != nil {
		t.Fatal(e)
	}
	if changed, e := WriteSplitLockedAdoption(roots[1], updates[roots[1]], true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	before := snapshot(t, roots[0])
	_, err := writeOwnersBatchWith(updates, false, nil, makeSplitLockedAdoptionPlan)
	errorCode(t, err, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, roots[0])) {
		t.Fatal("busy changed earlier root")
	}
	first, e := os.OpenFile(filepath.Join(roots[0], lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	if e := tryLock(first); e != nil {
		t.Fatal("earlier lock leaked", e)
	}
	unlock(first)
	first.Close()
	unlock(f)
	prepared := []int{}
	_, err = writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, stage string) error {
		if phase == "prepare" {
			prepared = append(prepared, ri)
		}
		if phase == "apply" && op == 0 {
			for _, root := range roots {
				lock, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
				if e != nil {
					t.Fatal(e)
				}
				e = tryLock(lock)
				if e == nil {
					unlock(lock)
				}
				lock.Close()
				if e == nil {
					t.Fatal("complete sorted lock set missing")
				}
			}
		}
		return nil
	}, makeSplitLockedAdoptionPlan)
	if err != nil || !reflect.DeepEqual(prepared, []int{0, 1}) {
		t.Fatal(err, prepared)
	}
	for _, entry := range []string{"manifest", "lock", "kotlin"} {
		root, u := splitLockedFixture(t)
		_, err := writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				switch entry {
				case "manifest":
					put(t, filepath.Join(root, manifestName), append(u.Previous.Manifest, ' '))
				case "lock":
					put(t, filepath.Join(root, nameLockFile), append(u.Previous.NameLock, ' '))
				case "kotlin":
					put(t, filepath.Join(root, u.Previous.Files[0].Path), []byte("edited after preflight"))
				}
			}
			return nil
		}, makeSplitLockedAdoptionPlan)
		code := "E_LOCKED_OWNER_ADOPTION_STALE"
		if entry == "lock" {
			code = "E_NAME_LOCK_MODIFIED"
		}
		if entry == "kotlin" {
			code = "E_OUTPUT_MODIFIED"
		}
		errorCode(t, err, code)
		if stages, _ := filepath.Glob(filepath.Join(root, stagePrefix+"*")); len(stages) != 0 {
			t.Fatal("staged stale candidate")
		}
	}
}

func TestSplitLockedAdoptionUnixSymlinksAliasRebindingAndDuplicateRoots(t *testing.T) {
	for _, kind := range []string{"manifest", "lock", "destination", "kotlin", "parent", "root"} {
		t.Run(kind, func(t *testing.T) {
			root, u := splitLockedFixture(t)
			path := root
			switch kind {
			case "manifest":
				path = filepath.Join(root, manifestName)
			case "lock":
				path = filepath.Join(root, nameLockFile)
			case "destination":
				path = filepath.Join(root, ownerNameLockFile("left"))
			case "kotlin":
				path = filepath.Join(root, u.Previous.Files[0].Path)
			case "parent":
				path = filepath.Dir(filepath.Join(root, u.Previous.Files[0].Path))
			}
			target := filepath.Join(root, nameLockFile)
			if kind != "destination" {
				target = path + "-original"
				if e := os.Rename(path, target); e != nil {
					t.Fatal(e)
				}
			}
			if e := os.Symlink(target, path); e != nil {
				t.Fatal(e)
			}
			for _, check := range []bool{true, false} {
				_, e := WriteSplitLockedAdoption(root, u, check)
				errorCode(t, e, "E_OUTPUT_PATH")
			}
		})
	}
	root, u := splitLockedFixture(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if e := os.Symlink(filepath.Dir(root), alias); e != nil {
		t.Fatal(e)
	}
	via := filepath.Join(alias, filepath.Base(root))
	_, e := writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u, via: u}, false, nil, makeSplitLockedAdoptionPlan)
	errorCode(t, e, "E_OUTPUT_ROOTS")
	s, e := ReadSplitLockedAdoptionState(via, u.Previous.OwnerNames)
	if e != nil {
		t.Fatal(e)
	}
	u.Previous = s
	// A canonical ancestor alias is accepted initially; rebinding after the source
	// read is refused even if another valid root has identical generation bytes.
	other, _ := splitLockedFixture(t)
	holder := t.TempDir()
	rebound := filepath.Join(holder, filepath.Base(root))
	if e := os.Rename(other, rebound); e != nil {
		t.Fatal(e)
	}
	if e := os.Remove(alias); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(holder, alias); e != nil {
		t.Fatal(e)
	}
	for _, check := range []bool{true, false} {
		_, e := WriteSplitLockedAdoption(via, u, check)
		errorCode(t, e, "E_LOCKED_OWNER_ADOPTION_STALE")
	}
}
