//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBatchAllLocksHeldUntilApplyAndSortedRoots(t *testing.T) {
	a, z := batchRoots(t)
	seen := []int{}
	updates := map[string]OwnerUpdate{z: batchUpdate("beta", true), a: batchUpdate("alpha", true)}
	_, err := writeOwnersBatch(updates, false, func(root int, phase string, index int, stage string) error {
		if phase == "prepare" {
			seen = append(seen, root)
		}
		if phase == "apply" && index == 0 {
			expected, e := canonicalRoot([]string{a, z}[root])
			if e != nil {
				t.Fatal(e)
			}
			if filepath.Dir(stage) != expected {
				t.Fatal("root ordering changed")
			}
			for _, path := range []string{a, z} {
				lock, e := os.OpenFile(filepath.Join(path, lockName), os.O_RDWR, 0600)
				if e != nil {
					t.Fatal(e)
				}
				e = tryLock(lock)
				if e == nil {
					unlock(lock)
				}
				_ = lock.Close()
				if e == nil {
					t.Fatal("batch did not hold complete root lock set")
				}
			}
		}
		return nil
	})
	if err != nil || !reflect.DeepEqual(seen, []int{0, 1}) {
		t.Fatal(err, seen)
	}
}
func TestBatchCanonicalParentAliasRefusedWithoutCreatingRoot(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if e := os.Mkdir(real, 0700); e != nil {
		t.Fatal(e)
	}
	alias := filepath.Join(base, "alias")
	if e := os.Symlink(real, alias); e != nil {
		t.Fatal(e)
	}
	u := batchUpdate("alpha", false)
	before := snapshot(t, real)
	_, e := WriteOwnersBatch(map[string]OwnerUpdate{filepath.Join(real, "out"): u, filepath.Join(alias, "out"): u}, false)
	errorCode(t, e, "E_OUTPUT_ROOTS")
	if !reflect.DeepEqual(before, snapshot(t, real)) {
		t.Fatal("alias refusal created output")
	}
}
