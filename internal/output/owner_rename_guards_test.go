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

func TestOwnerRenameOriginalInventoryRejectsSwapAndOrderedChain(t *testing.T) {
	root, a, u := renameFixture(t, true)
	u.Renames = map[string]string{"peer": "old", "old": "peer"}
	u.Generated = map[string]core.GenerationArtifacts{"old": a, "peer": a}
	_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false)
	errorCode(t, e, "E_OWNER_RENAME_INVALID")
	// If earlier a<-old modifies a working map, a subsequent old<-peer must still
	// reject old as occupied in the original inventory rather than accept a chain.
	u.Renames = map[string]string{"a": "old", "old": "peer"}
	u.Generated = map[string]core.GenerationArtifacts{"a": a, "old": a}
	before := snapshot(t, root)
	_, e = WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false)
	errorCode(t, e, "E_OWNER_RENAME_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("chain changed original inventory")
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
