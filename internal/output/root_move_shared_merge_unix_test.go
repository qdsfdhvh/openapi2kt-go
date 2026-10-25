//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveSharedMergeUnixCompleteLocks(t *testing.T) {
	for _, busySource := range []bool{false, true} {
		src, dst, s, _ := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
		s, e := BindRootMoveTarget(s, dst, src)
		if e != nil {
			t.Fatal(e)
		}
		root := dst
		if busySource {
			root = src
		}
		f, e0 := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
		if e0 != nil {
			t.Fatal(e0)
		}
		if e0 = tryLock(f); e0 != nil {
			t.Fatal(e0)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		if changed, e := MoveOutputRoot(src, dst, s, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		_, e = MoveOutputRoot(src, dst, s, false)
		errorCode(t, e, "E_OUTPUT_BUSY")
		unlock(f)
		_ = f.Close()
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("busy merge wrote")
		}
		_, e = moveOutputRoot(src, dst, s, false, func(_ int, phase string, op int, _ string) error {
			if phase == "apply" && op == 0 {
				for _, root := range []string{src, dst} {
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
						t.Fatal("merge missing complete cooperative locks")
					}
				}
			}
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
	}
}

func TestRootMoveSharedMergeUnixTargetAncestorRebind(t *testing.T) {
	src, dst, s, _ := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
	base := filepath.Dir(src)
	alias := filepath.Join(base, "alias")
	alternate := t.TempDir()
	if e := os.Symlink(base, alias); e != nil {
		t.Fatal(e)
	}
	raw := filepath.Join(alias, filepath.Base(dst))
	s, e := BindRootMoveOwnerTarget(s, raw, src, []string{"alpha"})
	if e != nil {
		t.Fatal(e)
	}
	a, b := snapshot(t, src), snapshot(t, dst)
	_, e = moveOutputSelection(src, raw, s, []string{"alpha"}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			if e := os.Remove(alias); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(alternate, alias); e != nil {
				t.Fatal(e)
			}
		}
		return nil
	})
	errorCode(t, e, "E_OUTPUT_ROOT_MOVE_STALE")
	if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
		t.Fatal("rebind merge wrote")
	}
}

func TestRootMoveSharedMergeUnixRollbackObstructionRetainsEvidence(t *testing.T) {
	src, dst, s, _ := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
	s, e := BindRootMoveTarget(s, dst, src)
	if e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, dst)
	path := s.Files[0].Path
	_, e = moveOutputRoot(src, dst, s, false, func(index int, phase string, _ int, _ string) error {
		if phase == "apply" && index == 1 {
			obstructLifecycleRollback(t, src, path)
			return errors.New("merge obstruction")
		}
		return nil
	})
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, src, path, s.Entries[path].Data)
	if !reflect.DeepEqual(before, snapshot(t, dst)) {
		t.Fatal("merge skipped unobstructed target rollback")
	}
}
