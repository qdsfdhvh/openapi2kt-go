//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNameLockOwnersBatchBusyReadonlyAndCompleteLockSetBeforePrepare(t *testing.T) {
	old, next := ownerBatchFixture(t)
	if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
		t.Fatal(e)
	}
	roots := sortedKeys(old)
	snap := map[string]map[string]snapshotEntry{}
	for _, root := range roots {
		snap[root] = snapshot(t, root)
	}
	holder, e := os.OpenFile(filepath.Join(roots[1], lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer holder.Close()
	if e := tryLock(holder); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadOwnerNameLock(roots[1], "beta"); e != nil {
		t.Fatal(e)
	}
	if changed, e := WriteNameLockedOwnersBatch(next, true); e != nil || !changed {
		t.Fatal(e)
	}
	_, failure := WriteNameLockedOwnersBatch(next, false)
	errorCode(t, failure, "E_OUTPUT_BUSY")
	for _, root := range roots {
		if !reflect.DeepEqual(snap[root], snapshot(t, root)) {
			t.Fatal("busy/check changed output")
		}
	}
	unlock(holder)
	prepared := 0
	_, failure = writeOwnersBatchWith(next, false, func(ri int, phase string, op int, stage string) error {
		if phase == "prepare" {
			prepared++
			for _, root := range roots {
				f, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
				if e != nil {
					t.Fatal(e)
				}
				if e := tryLock(f); e == nil {
					unlock(f)
					_ = f.Close()
					t.Fatal("prepare before complete root lock set")
				}
				_ = f.Close()
			}
		}
		return nil
	}, makeNameLockedOwnersPlan)
	if failure != nil || prepared != len(roots) {
		t.Fatal(prepared, failure)
	}
}
