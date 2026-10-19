package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestOwnerRenameIdentityDoesNotBypassGuardsOrOccurrencePreservation(t *testing.T) {
	for _, test := range []string{"occurrences", "source", "missing", "peerstate", "recovery"} {
		root, a, u := renameFixture(t, true)
		u.Renames = map[string]string{"old": "old"}
		u.Generated = map[string]core.GenerationArtifacts{"old": a}
		code := "E_OWNER_RENAME_CHANGED"
		switch test {
		case "occurrences":
			state, e := parseNameLockBytes(a.NameLock)
			if e != nil {
				t.Fatal(e)
			}
			entry := &state.Entries[0]
			i := len(entry.Occurrences) - 1
			entry.Occurrences[i] += "/synthetic"
			next, e := state.Encode()
			if e != nil {
				t.Fatal(e)
			}
			a.NameLock = next
			u.Generated["old"] = a
		case "source":
			a.Files[0].Content += "// drift"
			u.Generated["old"] = a
		case "missing":
			if e := os.Remove(filepath.Join(root, a.Files[0].Path)); e != nil {
				t.Fatal(e)
			}
			code = "E_OUTPUT_MISSING"
		case "peerstate":
			put(t, filepath.Join(root, ownerNameLockFile("peer")), []byte("bad"))
			code = "E_NAME_LOCK_MODIFIED"
		case "recovery":
			if e := os.Mkdir(filepath.Join(root, stagePrefix+"old"), 0700); e != nil {
				t.Fatal(e)
			}
			code = "E_RECOVERY_REQUIRED"
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("identity bypassed safety", test)
			}
		}
	}
}

func TestOwnerRenameOriginalInventorySwapAndOrderedChainGraduation(t *testing.T) {
	root, a, u := renameFixture(t, true)
	u.Renames = map[string]string{"peer": "old", "old": "peer"}
	u.Generated = map[string]core.GenerationArtifacts{"old": a, "peer": a}
	before := snapshot(t, root)
	// Preserve the exact original swap input. These two owners have identical
	// Kotlin and raw state, so the admitted permutation is a validated no-op.
	for _, check := range []bool{true, false} {
		if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check); e != nil || changed {
			t.Fatal("Original identical swap should be a no-op", changed, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("Identical owner swap rewrote output")
		}
	}
	// The exact former ordered-chain input now reads both source memberships
	// from the original inventory, even though one target is another source.
	u.Renames = map[string]string{"a": "old", "old": "peer"}
	u.Generated = map[string]core.GenerationArtifacts{"a": a, "old": a}
	if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("Chain check changed original inventory")
	}
	if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	after, e := ReadOwnerRenameState(root)
	if e != nil || !reflect.DeepEqual(after.Owners["a"], u.Previous.Owners["old"]) || !reflect.DeepEqual(after.Owners["old"], u.Previous.Owners["peer"]) || len(after.Owners) != 2 {
		t.Fatal("Chain did not preserve original memberships", after, e)
	}
}

func TestOwnerRenameExactByteCASWithRealPrelockSerializationChange(t *testing.T) {
	for _, test := range []string{"manifest", "payload-with-manifest", "payload-only"} {
		root, _, u := renameFixture(t, true)
		var intervening map[string]snapshotEntry
		_, e := writeOwnersBatchWith(map[string]OwnerRenameUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				if test == "manifest" {
					data := mustRead(t, filepath.Join(root, manifestName))
					put(t, filepath.Join(root, manifestName), append(data, ' '))
				} else {
					path := filepath.Join(root, ownerNameLockFile("peer"))
					data := append(mustRead(t, path), ' ')
					put(t, path, data)
					if test == "payload-with-manifest" {
						m, e := readManifestKind(root, true, true)
						if e != nil {
							t.Fatal(e)
						}
						(*m.NameLocks)["peer"] = digest(data)
						encoded, e2 := json.Marshal(m)
						if e2 != nil {
							t.Fatal(e2)
						}
						put(t, filepath.Join(root, manifestName), encoded)
					}
				}
				intervening = snapshot(t, root)
			}
			return nil
		}, makeOwnerRenamePlan)
		code := "E_OWNER_RENAME_STALE"
		if test == "payload-only" {
			code = "E_NAME_LOCK_MODIFIED"
		}
		errorCode(t, e, code)
		if !reflect.DeepEqual(intervening, snapshot(t, root)) {
			t.Fatal("CAS lost serialization-only intervening changes")
		}
	}
}
