//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func requireLifecycleLockSet(t testing.TB, roots []string, held bool) {
	t.Helper()
	for _, root := range roots {
		f, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		err := tryLock(f)
		if err == nil {
			unlock(f)
		}
		if e := f.Close(); e != nil {
			t.Fatal(e)
		}
		if (err != nil) != held {
			t.Fatal("unexpected cooperative lock set", root, held, err)
		}
	}
}

func TestNameLockLifecycleBatchSortedCompleteLockSetReplanAndAllStage(t *testing.T) {
	for _, noopFirst := range []bool{false, true} {
		old, next := lifecycleBatchFixture(t, NameLockDisable)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		if noopFirst {
			a := old[roots[0]].Generated["alpha"]
			next[roots[0]] = lifecycleOwnerUpdate("alpha", a, a.NameLock, NameLockReset)
		}
		for i, root := range roots {
			canonical, e := canonicalRoot(root)
			if e != nil {
				t.Fatal(e)
			}
			roots[i] = canonical
		}
		var plannedRoots, preparedRoots, appliedRoots []string
		plans := 0
		_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
			if phase == "prepare" {
				requireLifecycleLockSet(t, roots, true)
				preparedRoots = append(preparedRoots, roots[ri])
			}
			if phase == "apply" && index == 0 {
				requireLifecycleLockSet(t, roots, true)
				for _, root := range preparedRoots {
					stages, e := filepath.Glob(filepath.Join(root, stagePrefix+"*"))
					if e != nil || len(stages) != 1 {
						t.Fatal("apply began before all changed roots staged", root, stages, e)
					}
				}
				appliedRoots = append(appliedRoots, roots[ri])
			}
			return nil
		}, func(root string, u NameLockedOwnerUpdate) (*plan, *diagnostic.Diagnostic) {
			plannedRoots = append(plannedRoots, root)
			plans++
			if plans > len(roots) {
				requireLifecycleLockSet(t, roots, true)
			}
			return makeNameLockedOwnersPlan(root, u)
		})
		if e != nil {
			t.Fatal(e)
		}
		wantChanged := roots
		if noopFirst {
			wantChanged = roots[1:]
		}
		wantPlans := append(append([]string(nil), roots...), roots...)
		if !reflect.DeepEqual(plannedRoots, wantPlans) || !reflect.DeepEqual(preparedRoots, wantChanged) || !reflect.DeepEqual(appliedRoots, wantChanged) {
			t.Fatal("lifecycle did not follow sorted replan/prepare/apply order", plannedRoots, preparedRoots, appliedRoots)
		}
		requireLifecycleLockSet(t, roots, false)
	}
}

func TestNameLockLifecycleSingleBusyWriterAndReadonlyCheck(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		root := t.TempDir()
		req := singleLockRequest(t, core.DataClass, true)
		old := singleGenerated(t, req)
		seedLifecycleTest(t, root, false, old)
		next := lifecycleArtifacts(t, addedSinglePeer(t, req), action)
		before := snapshot(t, root)
		holder, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := tryLock(holder); e != nil {
			_ = holder.Close()
			t.Fatal(e)
		}
		if changed, e := WriteNameLockLifecycle(root, next, old.NameLock, action, true); e != nil || !changed {
			unlock(holder)
			_ = holder.Close()
			t.Fatal("single lifecycle check acquired busy writer lock", changed, e)
		}
		_, failure := WriteNameLockLifecycle(root, next, old.NameLock, action, false)
		unlock(holder)
		if e := holder.Close(); e != nil {
			t.Fatal(e)
		}
		errorCode(t, failure, "E_OUTPUT_BUSY")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("single busy/check lifecycle changed output")
		}
	}
}

func TestNameLockLifecycleBatchBusyCheckAndEarlierLockRelease(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		old, next := lifecycleBatchFixture(t, action)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		before := map[string]map[string]snapshotEntry{}
		for _, root := range roots {
			before[root] = snapshot(t, root)
		}
		holder, e := os.OpenFile(filepath.Join(roots[1], lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := tryLock(holder); e != nil {
			_ = holder.Close()
			t.Fatal(e)
		}
		if changed, e := WriteNameLockedOwnersBatch(next, true); e != nil || !changed {
			unlock(holder)
			_ = holder.Close()
			t.Fatal("busy writer blocked read-only lifecycle check", changed, e)
		}
		_, failure := WriteNameLockedOwnersBatch(next, false)
		errorCode(t, failure, "E_OUTPUT_BUSY")
		requireLifecycleLockSet(t, roots[:1], false)
		for _, root := range roots {
			if !reflect.DeepEqual(before[root], snapshot(t, root)) {
				t.Fatal("busy lifecycle modified output", root)
			}
		}
		unlock(holder)
		if e := holder.Close(); e != nil {
			t.Fatal(e)
		}
		requireLifecycleLockSet(t, roots, false)
	}
}
