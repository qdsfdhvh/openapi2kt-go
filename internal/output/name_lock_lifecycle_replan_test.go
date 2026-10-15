package output

import (
	"bytes"
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNameLockLifecycleSingleCASRechecksUnderLock(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		root := t.TempDir()
		req := singleLockRequest(t, core.DataClass, true)
		old := singleGenerated(t, req)
		seedLifecycleTest(t, root, false, old)
		next := lifecycleArtifacts(t, req, action)
		competingReq := addedSinglePeer(t, req)
		competingReq.PreviousNameLock = old.NameLock
		competing := singleGenerated(t, competingReq)
		var competitor map[string]snapshotEntry
		calls := 0
		_, e := writePlanned(root, false, func(root string) (*plan, *diagnostic.Diagnostic) {
			calls++
			p, e := makeNameLockPlan(root, next, old.NameLock, action)
			if e == nil && calls == 1 {
				// A cooperative writer commits after this preflight plan and before
				// the lifecycle writer's lock acquisition and second planning pass.
				if _, e := WriteNameLocked(root, competing, old.NameLock, false); e != nil {
					t.Fatal(e)
				}
				competitor = snapshot(t, root)
			}
			return p, e
		})
		errorCode(t, e, "E_NAME_LOCK_STALE")
		if calls != 2 || !reflect.DeepEqual(competitor, snapshot(t, root)) {
			t.Fatal("lifecycle skipped locked CAS or overwrote concurrent state", calls)
		}
	}
}

func TestNameLockLifecycleBatchCASRejectsBeforeAnyStage(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		old, next := lifecycleBatchFixture(t, action)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		before := snapshot(t, roots[0])
		var competitor map[string]snapshotEntry
		staged := false
		_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
			if phase == "prelock" {
				req := addedSinglePeer(t, ownerLockRequest(t, "beta", core.DataClass, true))
				req.PreviousNameLock = old[roots[1]].Generated["beta"].NameLock
				a := ownedGenerated(t, req)
				u := lockedUpdate([]string{"beta"}, map[string]core.GenerationArtifacts{"beta": a}, map[string][]byte{"beta": req.PreviousNameLock}, false)
				if _, e := WriteNameLockedOwners(roots[1], u, false); e != nil {
					t.Fatal(e)
				}
				competitor = snapshot(t, roots[1])
			}
			if phase == "prepare" {
				staged = true
			}
			return nil
		}, makeNameLockedOwnersPlan)
		errorCode(t, e, "E_NAME_LOCK_STALE")
		if staged || !reflect.DeepEqual(before, snapshot(t, roots[0])) || !reflect.DeepEqual(competitor, snapshot(t, roots[1])) {
			t.Fatal("batch staged early or lost concurrent output", action)
		}
	}
}

func TestNameLockLifecycleOwnersRetainedPeerReloadDuringLockedReplan(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		root := t.TempDir()
		ar, br := ownerLockRequest(t, "alpha", core.DataClass, true), ownerLockRequest(t, "beta", core.DataClass, true)
		a, b := ownedGenerated(t, ar), ownedGenerated(t, br)
		initial := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
		if _, e := WriteNameLockedOwners(root, initial, false); e != nil {
			t.Fatal(e)
		}
		ar = addedSinglePeer(t, ar)
		ar.Options.LockHoistedNames = action == NameLockReset
		an := ownedGenerated(t, ar)
		partial := lifecycleOwnerUpdate("alpha", an, a.NameLock, action)
		br = addedSinglePeer(t, br)
		br.PreviousNameLock = b.NameLock
		bn := ownedGenerated(t, br)
		var competitor map[string]snapshotEntry
		_, e := writeOwnersBatchWith(map[string]NameLockedOwnerUpdate{root: partial}, false, func(ri int, phase string, index int, stage string) error {
			if phase == "prelock" {
				u := lockedUpdate([]string{"beta"}, map[string]core.GenerationArtifacts{"beta": bn}, map[string][]byte{"beta": b.NameLock}, false)
				if _, e := WriteNameLockedOwners(root, u, false); e != nil {
					t.Fatal(e)
				}
				competitor = snapshot(t, root)
			}
			return nil
		}, makeNameLockedOwnersPlan)
		if e != nil {
			t.Fatal(e)
		}
		current, e := ReadOwnerNameLock(root, "beta")
		if e != nil || !bytes.Equal(current, bn.NameLock) {
			t.Fatal("lost intervening retained peer state", e)
		}
		after := snapshot(t, root)
		for _, f := range bn.Files {
			requireLifecycleFileUnchanged(t, competitor, after, f.Path)
		}
		if !reflect.DeepEqual(competitor[ownerNameLockFile("beta")], after[ownerNameLockFile("beta")]) {
			t.Fatal("rewrote retained peer metadata")
		}
		m, e := readManifestKind(root, true, true)
		if e != nil || m.Version != 4 || len(*m.Owners) != 2 {
			t.Fatal("lost retained peer inventory", m, e)
		}
	}
}

func TestNameLockLifecycleBatchRecoveryAppearingDuringPrelockIsRetained(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		old, next := lifecycleBatchFixture(t, action)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		first := snapshot(t, roots[0])
		var interrupted map[string]snapshotEntry
		staged := false
		_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
			if phase == "prelock" {
				path := filepath.Join(roots[1], stagePrefix+"synthetic-interruption")
				if e := os.Mkdir(path, 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(path, "transaction.json"), []byte("synthetic interrupted journal"))
				put(t, filepath.Join(path, "old-state"), old[roots[1]].Generated["beta"].NameLock)
				interrupted = snapshot(t, roots[1])
			}
			if phase == "prepare" {
				staged = true
			}
			return nil
		}, makeNameLockedOwnersPlan)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if staged || !reflect.DeepEqual(first, snapshot(t, roots[0])) || !reflect.DeepEqual(interrupted, snapshot(t, roots[1])) {
			t.Fatal("locked lifecycle missed or erased new recovery evidence")
		}
	}
}
