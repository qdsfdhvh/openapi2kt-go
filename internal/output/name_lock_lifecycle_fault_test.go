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

func lifecycleFaultArtifacts(t testing.TB, req core.GenerationRequest, action NameLockAction) (core.GenerationArtifacts, core.GenerationArtifacts) {
	t.Helper()
	old := singleGenerated(t, req)
	next := lifecycleArtifacts(t, addedSinglePeer(t, req), action)
	old.Files = append(old.Files, core.GeneratedFile{Path: "stale/Synthetic.kt", Content: "// synthetic old owned sentinel"})
	next.Files = append(next.Files, core.GeneratedFile{Path: "new/deep/Synthetic.kt", Content: "// synthetic new owned sentinel"})
	return old, next
}

func checkLifecycleJournal(t testing.TB, stage, stateFile string, action NameLockAction) journal {
	t.Helper()
	var j journal
	if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
		t.Fatal(e)
	}
	if j.Operations[len(j.Operations)-1].Path != manifestName {
		t.Fatal("manifest must commit last")
	}
	found := false
	for _, op := range j.Operations {
		if op.Path == stateFile {
			found = true
			if !op.HadOld || (op.Staged == nil) != (action == NameLockDisable) {
				t.Fatal("state is not a journaled owned replacement/removal", op)
			}
		}
	}
	if !found {
		t.Fatal("lifecycle metadata is outside transaction")
	}
	return j
}

func TestNameLockLifecycleSingleEveryCommitBoundaryRollback(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		t.Run(string(action), func(t *testing.T) {
			root := t.TempDir()
			old, next := lifecycleFaultArtifacts(t, singleLockRequest(t, core.DataClass, true), action)
			seedLifecycleTest(t, root, false, old)
			put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten"))
			before := snapshot(t, root)
			p, e := makeNameLockPlan(root, next, old.NameLock, action)
			if e != nil {
				t.Fatal(e)
			}
			operations := len(p.Writes) + len(p.Removals) + 1
			if operations < 5 {
				t.Fatal("insufficient replacement/creation/removal/metadata/manifest coverage")
			}
			for boundary := 0; boundary < operations; boundary++ {
				e := commit(root, p, func(index int, stage string) error {
					checkLifecycleJournal(t, stage, nameLockFile, action)
					if index == boundary {
						return errors.New("synthetic lifecycle boundary")
					}
					return nil
				})
				errorCode(t, e, "E_OUTPUT_COMMIT")
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("single rollback lost bytes/modes/mtimes or created directories", action, boundary)
				}
			}
			t.Log("injected every", operations, "commit boundary")
		})
	}
}

func TestNameLockLifecycleResetFailureAfterMetadataBackup(t *testing.T) {
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		old, next := lifecycleFaultArtifacts(t, singleLockRequest(t, core.DataClass, true), NameLockReset)
		stateFile := seedLifecycleTest(t, root, named, old)
		before := snapshot(t, root)
		p, e := makeNameLockPlan(root, next, old.NameLock, NameLockReset)
		if named {
			p, e = makeNameLockedOwnersPlan(root, lifecycleOwnerUpdate("alpha", next, old.NameLock, NameLockReset))
		}
		if e != nil {
			t.Fatal(e)
		}
		e = commit(root, p, func(index int, stage string) error {
			j := checkLifecycleJournal(t, stage, stateFile, NameLockReset)
			if j.Operations[index].Path == stateFile {
				// The old destination moves to backup before the missing staged
				// replacement fails, so this covers restoration after that move.
				if e := os.Remove(filepath.Join(stage, *j.Operations[index].Staged)); e != nil {
					t.Fatal(e)
				}
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("failed replacement lost old metadata", named)
		}
	}
}

func obstructLifecycleRollback(t testing.TB, root, stateFile string) {
	t.Helper()
	path := filepath.Join(root, stateFile)
	if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
		t.Fatal(e)
	}
	if e := os.Mkdir(path, 0700); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(path, "obstruction"), []byte("synthetic rollback obstruction"))
}

func requireLifecycleBackup(t testing.TB, root, stateFile string, old []byte) {
	t.Helper()
	stages, e := filepath.Glob(filepath.Join(root, stagePrefix+"*"))
	if e != nil || len(stages) != 1 {
		t.Fatal("missing retained stage", stages, e)
	}
	var j journal
	if e := json.Unmarshal(mustRead(t, filepath.Join(stages[0], "transaction.json")), &j); e != nil {
		t.Fatal(e)
	}
	found := false
	for _, op := range j.Operations {
		if op.Path == stateFile {
			found = true
			if string(mustRead(t, filepath.Join(stages[0], op.Backup))) != string(old) {
				t.Fatal("lost old state backup")
			}
		}
	}
	if !found {
		t.Fatal("metadata missing from retained journal")
	}
}

func TestNameLockLifecycleSingleRollbackObstructionKeepsEvidence(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		root := t.TempDir()
		old, next := lifecycleFaultArtifacts(t, singleLockRequest(t, core.DataClass, true), action)
		seedLifecycleTest(t, root, false, old)
		p, e := makeNameLockPlan(root, next, old.NameLock, action)
		if e != nil {
			t.Fatal(e)
		}
		e = commit(root, p, func(index int, stage string) error {
			j := checkLifecycleJournal(t, stage, nameLockFile, action)
			if j.Operations[index].Path == manifestName {
				obstructLifecycleRollback(t, root, nameLockFile)
				return errors.New("synthetic lifecycle rollback obstruction")
			}
			return nil
		})
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		requireLifecycleBackup(t, root, nameLockFile, old.NameLock)
		before := snapshot(t, root)
		_, e = ReadNameLock(root)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockLifecycle(root, next, old.NameLock, action, check)
			errorCode(t, e, "E_RECOVERY_REQUIRED")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("lifecycle erased failed rollback evidence")
			}
		}
	}
}

func lifecycleBatchFixture(t testing.TB, action NameLockAction) (map[string]NameLockedOwnerUpdate, map[string]NameLockedOwnerUpdate) {
	t.Helper()
	old, next := map[string]NameLockedOwnerUpdate{}, map[string]NameLockedOwnerUpdate{}
	base := t.TempDir()
	for _, owner := range []string{"alpha", "beta"} {
		root := filepath.Join(base, owner)
		a, b := lifecycleFaultArtifacts(t, ownerLockRequest(t, owner, core.DataClass, true), action)
		old[root] = lockedUpdate([]string{owner}, map[string]core.GenerationArtifacts{owner: a}, map[string][]byte{owner: nil}, true)
		next[root] = lifecycleOwnerUpdate(owner, b, a.NameLock, action)
	}
	return old, next
}

func TestNameLockLifecycleBatchEveryBoundaryAndPrepareFailure(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		t.Run(string(action), func(t *testing.T) {
			old, next := lifecycleBatchFixture(t, action)
			if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
				t.Fatal(e)
			}
			roots := sortedKeys(old)
			before := map[string]map[string]snapshotEntry{}
			for _, root := range roots {
				put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic unowned"))
				before[root] = snapshot(t, root)
			}
			assertRestored := func() {
				t.Helper()
				for _, root := range roots {
					if !reflect.DeepEqual(before[root], snapshot(t, root)) {
						t.Fatal("batch rollback changed bytes/modes/mtimes/Kotlin/state", root)
					}
				}
			}
			operations := 0
			for rootIndex, root := range roots {
				p, e := makeNameLockedOwnersPlan(root, next[root])
				if e != nil {
					t.Fatal(e)
				}
				count := len(p.Writes) + len(p.Removals) + 1
				operations += count
				for boundary := 0; boundary < count; boundary++ {
					_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
						if phase == "apply" {
							checkLifecycleJournal(t, stage, ownerNameLockFile(filepath.Base(roots[ri])), action)
							if ri == rootIndex && index == boundary {
								return errors.New("synthetic lifecycle batch boundary")
							}
						}
						return nil
					}, makeNameLockedOwnersPlan)
					errorCode(t, e, "E_OUTPUT_COMMIT")
					assertRestored()
				}
				_, e = writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
					if phase == "prepare" && ri == rootIndex {
						return errors.New("synthetic lifecycle preparation failure")
					}
					return nil
				}, makeNameLockedOwnersPlan)
				errorCode(t, e, "E_OUTPUT_IO")
				assertRestored()
			}
			if operations < 10 {
				t.Fatal("insufficient batch boundary coverage")
			}
			t.Log("injected", operations, "commit boundaries and every root preparation")
			if changed, e := WriteNameLockedOwnersBatch(next, true); e != nil || !changed {
				t.Fatal(changed, e)
			}
			assertRestored()
			if changed, e := WriteNameLockedOwnersBatch(next, false); e != nil || !changed {
				t.Fatal(changed, e)
			}
		})
	}
}

func TestNameLockLifecycleBatchRollbackObstructionRestoresOtherRoot(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		old, next := lifecycleBatchFixture(t, action)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		peerBefore := snapshot(t, roots[1])
		_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
			if phase == "apply" && ri == 1 {
				j := checkLifecycleJournal(t, stage, ownerNameLockFile("beta"), action)
				if j.Operations[index].Path == manifestName {
					obstructLifecycleRollback(t, roots[0], ownerNameLockFile("alpha"))
					return errors.New("synthetic obstructed batch lifecycle rollback")
				}
			}
			return nil
		}, makeNameLockedOwnersPlan)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(peerBefore, snapshot(t, roots[1])) {
			t.Fatal("obstructed first root prevented second root restoration")
		}
		requireLifecycleBackup(t, roots[0], ownerNameLockFile("alpha"), old[roots[0]].Generated["alpha"].NameLock)
		before := snapshot(t, roots[0])
		_, e = ReadOwnerNameLock(roots[0], "alpha")
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwnersBatch(next, check)
			errorCode(t, e, "E_RECOVERY_REQUIRED")
			if !reflect.DeepEqual(before, snapshot(t, roots[0])) || !reflect.DeepEqual(peerBefore, snapshot(t, roots[1])) {
				t.Fatal("batch lifecycle erased failed rollback evidence")
			}
		}
	}
}
