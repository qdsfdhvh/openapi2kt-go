package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func renameFixture(t testing.TB, locked bool) (string, core.GenerationArtifacts, OwnerRenameUpdate) {
	t.Helper()
	root := t.TempDir()
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, locked))
	a.Files = append(a.Files, core.GeneratedFile{Path: "shared/Runtime.kt", Content: "// synthetic shared runtime"})
	u := lockedUpdate([]string{"old", "peer"}, map[string]core.GenerationArtifacts{"old": a, "peer": a}, nil, true)
	if locked {
		u.PreviousNameLocks = map[string][]byte{"old": nil, "peer": nil}
	}
	if _, e := WriteNameLockedOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	state, e := ReadOwnerRenameState(root)
	if e != nil {
		t.Fatal(e)
	}
	return root, a, OwnerRenameUpdate{Previous: state, Renames: map[string]string{"new": "old"}, Generated: map[string]core.GenerationArtifacts{"new": a}}
}

func TestOwnerRenameMembershipStateReadOnlyAndNoop(t *testing.T) {
	for _, locked := range []bool{false, true} {
		root, a, u := renameFixture(t, locked)
		put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten"))
		before := snapshot(t, root)
		if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("rename check mutated output")
		}
		if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		after := snapshot(t, root)
		for _, f := range a.Files {
			if !reflect.DeepEqual(before[f.Path], after[f.Path]) {
				t.Fatal("Kotlin changed", f.Path)
			}
		}
		if !reflect.DeepEqual(before["Handwritten.kt"], after["Handwritten.kt"]) {
			t.Fatal("unowned changed")
		}
		var m manifest
		if e := json.Unmarshal(mustRead(t, filepath.Join(root, manifestName)), &m); e != nil {
			t.Fatal(e)
		}
		if _, exists := (*m.Owners)["old"]; exists || !reflect.DeepEqual((*m.Owners)["new"], (*m.Owners)["peer"]) {
			t.Fatal("wrong membership", m)
		}
		if locked {
			if m.Version != 4 || (*m.NameLocks)["new"] != digest(a.NameLock) || string(mustRead(t, filepath.Join(root, ownerNameLockFile("new")))) != string(a.NameLock) {
				t.Fatal("state not preserved")
			}
			if _, e := os.Stat(filepath.Join(root, ownerNameLockFile("old"))); !os.IsNotExist(e) {
				t.Fatal("old state survived", e)
			}
			if !reflect.DeepEqual(before[ownerNameLockFile("peer")], after[ownerNameLockFile("peer")]) {
				t.Fatal("peer state changed")
			}
		} else if m.Version != 2 {
			t.Fatal("version changed")
		}
		state, e := ReadOwnerRenameState(root)
		if e != nil {
			t.Fatal(e)
		}
		u.Previous = state
		u.Renames = map[string]string{"new": "new"}
		before = snapshot(t, root)
		for _, check := range []bool{true, false} {
			if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check); e != nil || changed {
				t.Fatal("identity rename", changed, e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("noop rewrote bytes/mtime")
			}
		}
	}
}

func TestOwnerRenameRejectsUnsafeOrChangedCandidates(t *testing.T) {
	for _, test := range []string{"unknown", "occupied", "candidate", "membership", "disabled", "state", "missing", "modified", "peerstate", "collision", "recovery", "manifest", "stale"} {
		t.Run(test, func(t *testing.T) {
			root, a, u := renameFixture(t, true)
			code := "E_OWNER_RENAME_INVALID"
			switch test {
			case "unknown":
				u.Renames["new"] = "absent"
			case "occupied":
				u.Renames = map[string]string{"peer": "old"}
				u.Generated = map[string]core.GenerationArtifacts{"peer": a}
			case "candidate":
				a.Files[0].Content += "\n// drift"
				u.Generated["new"] = a
				code = "E_OWNER_RENAME_CHANGED"
			case "membership":
				a.Files = a.Files[1:]
				u.Generated["new"] = a
				code = "E_OWNER_RENAME_CHANGED"
			case "disabled":
				a.NameLock = nil
				u.Generated["new"] = a
				code = "E_OWNER_RENAME_CHANGED"
			case "state":
				a.NameLock = singleGenerated(t, addedSinglePeer(t, singleLockRequest(t, core.DataClass, true))).NameLock
				u.Generated["new"] = a
				code = "E_OWNER_RENAME_CHANGED"
			case "missing":
				if e := os.Remove(filepath.Join(root, a.Files[0].Path)); e != nil {
					t.Fatal(e)
				}
				code = "E_OUTPUT_MISSING"
			case "modified":
				put(t, filepath.Join(root, a.Files[0].Path), []byte("edited"))
				code = "E_OUTPUT_MODIFIED"
			case "peerstate":
				put(t, filepath.Join(root, ownerNameLockFile("peer")), []byte("edited"))
				code = "E_NAME_LOCK_MODIFIED"
			case "collision":
				put(t, filepath.Join(root, ownerNameLockFile("new")), a.NameLock)
				code = "E_OUTPUT_COLLISION"
			case "recovery":
				if e := os.Mkdir(filepath.Join(root, stagePrefix+"legacy"), 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(root, stagePrefix+"legacy", "transaction.json"), []byte("malformed"))
				code = "E_RECOVERY_REQUIRED"
			case "manifest":
				put(t, filepath.Join(root, manifestName), []byte(`{"version":4,"version":4}`))
				code = "E_MANIFEST_INVALID"
			case "stale":
				u.Previous.Manifest = append(u.Previous.Manifest, ' ')
				code = "E_OWNER_RENAME_STALE"
			}
			before := snapshot(t, root)
			for _, check := range []bool{true, false} {
				_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("rejection mutated old output", test)
				}
			}
		})
	}
}

func TestOwnerRenameBatchEveryFaultBoundary(t *testing.T) {
	updates := map[string]OwnerRenameUpdate{}
	before := map[string]map[string]snapshotEntry{}
	for i := 0; i < 2; i++ {
		root, _, u := renameFixture(t, true)
		updates[root] = u
		before[root] = snapshot(t, root)
	}
	assertOld := func() {
		t.Helper()
		for root, old := range before {
			if !reflect.DeepEqual(old, snapshot(t, root)) {
				t.Fatal("rollback lost old bytes/mode/mtime", root)
			}
		}
	}
	for ri, root := range sortedKeys(updates) {
		p, e := makeOwnerRenamePlan(root, updates[root])
		if e != nil {
			t.Fatal(e)
		}
		if len(p.Writes) != 1 || len(p.Removals) != 1 {
			t.Fatal("rename should move only metadata", p)
		}
		for op := 0; op < 3; op++ {
			_, e := writeOwnersBatchWith(updates, false, func(index int, phase string, operation int, stage string) error {
				if phase == "apply" {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if j.Operations[len(j.Operations)-1].Path != manifestName {
						t.Fatal("manifest not last")
					}
				}
				if index == ri && phase == "apply" && operation == op {
					return errors.New("synthetic fault")
				}
				return nil
			}, makeOwnerRenamePlan)
			errorCode(t, e, "E_OUTPUT_COMMIT")
			assertOld()
		}
		_, e = writeOwnersBatchWith(updates, false, func(index int, phase string, _ int, _ string) error {
			if index == ri && phase == "prepare" {
				return errors.New("synthetic preparation fault")
			}
			return nil
		}, makeOwnerRenamePlan)
		errorCode(t, e, "E_OUTPUT_IO")
		assertOld()
	}
}

func TestOwnerRenameManifestCASUnderLock(t *testing.T) {
	root, a, u := renameFixture(t, true)
	var intervening map[string]snapshotEntry
	_, e := writeOwnersBatchWith(map[string]OwnerRenameUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			peer := lockedUpdate([]string{"third"}, map[string]core.GenerationArtifacts{"third": a}, map[string][]byte{"third": nil}, false)
			if _, e := WriteNameLockedOwners(root, peer, false); e != nil {
				t.Fatal(e)
			}
			intervening = snapshot(t, root)
		}
		return nil
	}, makeOwnerRenamePlan)
	errorCode(t, e, "E_OWNER_RENAME_STALE")
	if !reflect.DeepEqual(intervening, snapshot(t, root)) {
		t.Fatal("stale rename lost intervening owner")
	}
}
