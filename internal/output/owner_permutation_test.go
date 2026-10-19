package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/core"
)

func ownerPermutationFixture(t testing.TB, first, second bool) (string, OwnerRenameUpdate, map[string]core.GenerationArtifacts) {
	t.Helper()
	root := t.TempDir()
	artifacts := map[string]core.GenerationArtifacts{}
	previous := map[string][]byte{}
	for owner, locked := range map[string]bool{"alpha": first, "beta": second, "peer": first} {
		artifacts[owner] = ownedGenerated(t, ownerLockRequest(t, owner, core.DataClass, locked))
		if locked {
			previous[owner] = nil
		}
	}
	if _, e := WriteNameLockedOwners(root, lockedUpdate([]string{"alpha", "beta", "peer"}, artifacts, previous, true), false); e != nil {
		t.Fatal(e)
	}
	if first {
		// Preserve valid noncanonical raw state, rather than only decoded objects.
		path := filepath.Join(root, ownerNameLockFile("alpha"))
		raw := append([]byte("\n"), mustRead(t, path)...)
		put(t, path, raw)
		m, e := readManifestKind(root, true, true)
		if e != nil {
			t.Fatal(e)
		}
		(*m.NameLocks)["alpha"] = digest(raw)
		encoded, _ := json.Marshal(m)
		put(t, filepath.Join(root, manifestName), encoded)
	}
	path := filepath.Join(root, artifacts["alpha"].Files[0].Path)
	stamp := time.Unix(1700000000, 0)
	if e := os.Chmod(path, 0640); e != nil {
		t.Fatal(e)
	}
	if e := os.Chtimes(path, stamp, stamp); e != nil {
		t.Fatal(e)
	}
	state, e := ReadOwnerRenameState(root)
	if e != nil {
		t.Fatal(e)
	}
	u := OwnerRenameUpdate{Previous: state, Renames: map[string]string{"alpha": "beta", "beta": "alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": artifacts["beta"], "beta": artifacts["alpha"]}}
	return root, u, artifacts
}

func TestOwnerPermutationOriginalMembershipRawStateAndMixedModes(t *testing.T) {
	for _, modes := range [][2]bool{{false, false}, {true, true}, {true, false}, {false, true}} {
		for _, mappings := range []map[string]string{
			{"alpha": "beta", "beta": "alpha"},
			{"alpha": "beta", "omega": "alpha"},
			{"alpha": "peer", "beta": "alpha", "peer": "beta"},
		} {
			root, u, artifacts := ownerPermutationFixture(t, modes[0], modes[1])
			u.Renames = mappings
			u.Generated = map[string]core.GenerationArtifacts{}
			for next, old := range mappings {
				u.Generated[next] = artifacts[old]
			}
			put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic unowned"))
			before := snapshot(t, root)
			if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, true); e != nil || !changed {
				t.Fatal("Permutation check", modes, mappings, changed, e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("Permutation check mutated output")
			}
			if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil || !changed {
				t.Fatal("Permutation apply", modes, mappings, changed, e)
			}
			after := snapshot(t, root)
			state, e := ReadOwnerRenameState(root)
			if e != nil || state.Version != u.Previous.Version {
				t.Fatal("Permutation changed manifest version", e)
			}
			for next, old := range mappings {
				if !reflect.DeepEqual(state.Owners[next], u.Previous.Owners[old]) || !bytes.Equal(state.NameLocks[next], u.Previous.NameLocks[old]) {
					t.Fatal("Lost original source membership/raw lock", modes, mappings, next)
				}
			}
			for _, owner := range []string{"alpha", "beta", "peer", "omega"} {
				if len(state.NameLocks[owner]) == 0 {
					if _, exists := after[ownerNameLockFile(owner)]; exists {
						t.Fatal("Unlocked or removed owner retained managed state", owner)
					}
				}
			}
			for path, old := range before {
				if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(old, after[path]) {
					t.Fatal("Permutation rewrote Kotlin/unowned bytes/mode/mtime", path)
				}
			}
			if _, selected := mappings["peer"]; !selected {
				if !reflect.DeepEqual(state.Owners["peer"], u.Previous.Owners["peer"]) || !reflect.DeepEqual(before[ownerNameLockFile("peer")], after[ownerNameLockFile("peer")]) {
					t.Fatal("Unselected peer changed")
				}
			}
		}
	}
}

func TestOwnerPermutationEveryOverwriteFaultAndCrossRootRollback(t *testing.T) {
	updates := map[string]OwnerRenameUpdate{}
	before := map[string]map[string]snapshotEntry{}
	for i := 0; i < 2; i++ {
		root, u, _ := ownerPermutationFixture(t, true, true)
		updates[root], before[root] = u, snapshot(t, root)
	}
	for ri, root := range sortedKeys(updates) {
		p, e := makeOwnerRenamePlan(root, updates[root])
		if e != nil || len(p.Writes) != 2 || len(p.Removals) != 0 {
			t.Fatal("Swapped destinations must be written once and never removed", p, e)
		}
		for operation := 0; operation < 3; operation++ {
			_, e := writeOwnersBatchWith(updates, false, func(index int, phase string, op int, stage string) error {
				if phase == "apply" {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if j.Operations[len(j.Operations)-1].Path != manifestName {
						t.Fatal("Permutation manifest must remain last")
					}
				}
				if index == ri && phase == "apply" && op == operation {
					return errors.New("synthetic overwrite fault")
				}
				return nil
			}, makeOwnerRenamePlan)
			errorCode(t, e, "E_OUTPUT_COMMIT")
			for root, old := range before {
				if !reflect.DeepEqual(old, snapshot(t, root)) {
					t.Fatal("Swap rollback lost original bytes/mode/mtime", root, ri, operation)
				}
			}
		}
	}
}
