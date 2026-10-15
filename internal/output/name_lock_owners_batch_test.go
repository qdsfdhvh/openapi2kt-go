package output

import (
	"encoding/json"
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func ownerBatchFixture(t testing.TB) (map[string]NameLockedOwnerUpdate, map[string]NameLockedOwnerUpdate) {
	t.Helper()
	old, next := map[string]NameLockedOwnerUpdate{}, map[string]NameLockedOwnerUpdate{}
	base := t.TempDir()
	for _, owner := range []string{"alpha", "beta"} {
		root := filepath.Join(base, owner)
		r := ownerLockRequest(t, owner, core.DataClass, true)
		a := ownedGenerated(t, r)
		old[root] = lockedUpdate([]string{owner}, map[string]core.GenerationArtifacts{owner: a}, map[string][]byte{owner: nil}, true)
		r = addedSinglePeer(t, r)
		r.PreviousNameLock = a.NameLock
		b := ownedGenerated(t, r)
		next[root] = lockedUpdate([]string{owner}, map[string]core.GenerationArtifacts{owner: b}, map[string][]byte{owner: a.NameLock}, true)
	}
	return old, next
}
func TestNameLockOwnersBatchEveryBoundaryAndPrepareFailure(t *testing.T) {
	old, next := ownerBatchFixture(t)
	if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
		t.Fatal(e)
	}
	roots := sortedKeys(old)
	before := map[string]map[string]snapshotEntry{}
	for _, root := range roots {
		put(t, filepath.Join(root, "Handwritten.kt"), []byte("synthetic unowned"))
		before[root] = snapshot(t, root)
	}
	operations := 0
	for rootIndex, root := range roots {
		p, e := makeNameLockedOwnersPlan(root, next[root])
		if e != nil {
			t.Fatal(e)
		}
		n := len(p.Writes) + len(p.Removals) + 1
		operations += n
		for op := 0; op < n; op++ {
			_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
				if phase == "apply" {
					var j journal
					if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); err != nil {
						t.Fatal(err)
					}
					if j.Operations[len(j.Operations)-1].Path != manifestName {
						t.Fatal("manifest not last")
					}
					if ri == rootIndex && index == op {
						return errors.New("synthetic named lock batch boundary")
					}
				}
				return nil
			}, makeNameLockedOwnersPlan)
			errorCode(t, e, "E_OUTPUT_COMMIT")
			for _, r := range roots {
				if !reflect.DeepEqual(before[r], snapshot(t, r)) {
					t.Fatal("batch lost state/manifest/Kotlin bytes-mode-mtime", riLabel(rootIndex, op), r)
				}
			}
		}
	}
	_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
		if ri == 1 && phase == "prepare" {
			return errors.New("synthetic final prepare failure")
		}
		return nil
	}, makeNameLockedOwnersPlan)
	errorCode(t, e, "E_OUTPUT_IO")
	for _, r := range roots {
		if !reflect.DeepEqual(before[r], snapshot(t, r)) {
			t.Fatal("prepare failure changed output")
		}
	}
	if operations < 6 {
		t.Fatal("too few fault boundaries")
	}
	t.Log("injected", operations, "batch operations across", len(roots), "roots plus final prepare failure")
	if changed, e := WriteNameLockedOwnersBatch(next, true); e != nil || !changed {
		t.Fatal(e)
	}
	for _, r := range roots {
		if !reflect.DeepEqual(before[r], snapshot(t, r)) {
			t.Fatal("batch check wrote")
		}
	}
	if _, e := WriteNameLockedOwnersBatch(next, false); e != nil {
		t.Fatal(e)
	}
	for _, root := range roots {
		owner := filepath.Base(root)
		u := next[root]
		u.PreviousNameLocks[owner] = u.Generated[owner].NameLock
		next[root] = u
	}
	stable := map[string]map[string]snapshotEntry{}
	for _, root := range roots {
		stable[root] = snapshot(t, root)
	}
	if changed, e := WriteNameLockedOwnersBatch(next, false); e != nil || changed {
		t.Fatal(e)
	}
	for _, r := range roots {
		if !reflect.DeepEqual(stable[r], snapshot(t, r)) {
			t.Fatal("batch noop wrote")
		}
	}
}
func riLabel(root, op int) [2]int { return [2]int{root, op} }
func TestNameLockOwnersBatchCASAfterPreflightBeforeAnyStage(t *testing.T) {
	old, next := ownerBatchFixture(t)
	if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
		t.Fatal(e)
	}
	roots := sortedKeys(old)
	before := snapshot(t, roots[0])
	var competing map[string]snapshotEntry
	_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
		if phase == "prelock" {
			if _, e := WriteNameLockedOwners(roots[1], next[roots[1]], false); e != nil {
				t.Fatal(e)
			}
			competing = snapshot(t, roots[1])
		}
		return nil
	}, makeNameLockedOwnersPlan)
	errorCode(t, e, "E_NAME_LOCK_STALE")
	if !reflect.DeepEqual(before, snapshot(t, roots[0])) || !reflect.DeepEqual(competing, snapshot(t, roots[1])) {
		t.Fatal("locked CAS overwrote competitor or staged first root")
	}
}
func TestNameLockOwnersBatchRecoveryObstructionRetainsNameBackup(t *testing.T) {
	old, next := ownerBatchFixture(t)
	if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
		t.Fatal(e)
	}
	roots := sortedKeys(old)
	_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
		if ri == 1 && phase == "apply" && index == 0 {
			path := filepath.Join(roots[0], ownerNameLockFile("alpha"))
			if e := os.Remove(path); e != nil {
				t.Fatal(e)
			}
			if e := os.Mkdir(path, 0700); e != nil {
				t.Fatal(e)
			}
			return errors.New("synthetic obstructed state rollback")
		}
		return nil
	}, makeNameLockedOwnersPlan)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	stages, _ := filepath.Glob(filepath.Join(roots[0], stagePrefix+"*"))
	if len(stages) != 1 {
		t.Fatal("missing retained journal", stages)
	}
	var j journal
	_ = json.Unmarshal(mustRead(t, filepath.Join(stages[0], "transaction.json")), &j)
	found := false
	for _, op := range j.Operations {
		if op.Path == ownerNameLockFile("alpha") {
			found = true
			if string(mustRead(t, filepath.Join(stages[0], op.Backup))) != string(old[roots[0]].Generated["alpha"].NameLock) {
				t.Fatal("lost old state backup")
			}
		}
	}
	if !found {
		t.Fatal("state absent from retained journal")
	}
	_, e = ReadOwnerNameLock(roots[0], "alpha")
	errorCode(t, e, "E_RECOVERY_REQUIRED")
}
