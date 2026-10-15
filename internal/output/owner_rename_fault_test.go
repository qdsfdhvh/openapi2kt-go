package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOwnerRenameInvalidMissingMetadataAndUnsupportedVersions(t *testing.T) {
	for _, test := range []string{"missing", "invalid", "unsupported", "v1", "v3", "unlocked-old-destination"} {
		root, _, u := renameFixture(t, true)
		m, e := readManifestKind(root, true, true)
		if e != nil {
			t.Fatal(e)
		}
		code := "E_NAME_LOCK_MISSING"
		switch test {
		case "missing":
			if e := os.Remove(filepath.Join(root, ownerNameLockFile("old"))); e != nil {
				t.Fatal(e)
			}
		case "invalid", "unsupported":
			state := []byte(`{"version":1,"profile":{},"entries":[]}`)
			code = "E_NAME_LOCK_INVALID"
			if test == "unsupported" {
				state = []byte(`{"version":99,"profile":{"package":"org.example.lock","suffix":"","qualified":false},"entries":[]}`)
				code = "E_NAME_LOCK_VERSION"
			}
			put(t, filepath.Join(root, ownerNameLockFile("old")), state)
			(*m.NameLocks)["old"] = digest(state)
			data, e := json.Marshal(m)
			if e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(root, manifestName), data)
		case "v1":
			m.Version = 1
			m.Owners = nil
			m.NameLocks = nil
			data, _ := json.Marshal(m)
			put(t, filepath.Join(root, manifestName), data)
			code = "E_MANIFEST_OWNER_MIGRATION"
		case "v3":
			m.Version = 3
			m.Owners = nil
			m.NameLocks = nil
			hash := digest([]byte("synthetic"))
			m.NameLock = &hash
			data, _ := json.Marshal(m)
			put(t, filepath.Join(root, manifestName), data)
			code = "E_MANIFEST_VERSION"
		case "unlocked-old-destination":
			root, _, u = renameFixture(t, false)
			put(t, filepath.Join(root, ownerNameLockFile("old")), []byte("unowned state"))
			code = "E_OUTPUT_COLLISION"
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, check)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("unsafe metadata removed", test)
			}
		}
	}
	absent := filepath.Join(t.TempDir(), "absent")
	_, e := ReadOwnerRenameState(absent)
	errorCode(t, e, "E_OWNER_RENAME_INVALID")
	if _, e := os.Stat(absent); !os.IsNotExist(e) {
		t.Fatal("rename read created root")
	}
}

func TestOwnerRenameBatchRollbackObstructionRetainsEvidence(t *testing.T) {
	updates := map[string]OwnerRenameUpdate{}
	for i := 0; i < 2; i++ {
		root, _, u := renameFixture(t, true)
		updates[root] = u
	}
	roots := sortedKeys(updates)
	peer := snapshot(t, roots[1])
	old := updates[roots[0]].Previous.NameLocks["old"]
	_, e := writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, stage string) error {
		if phase == "apply" && ri == 1 {
			var j journal
			if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
				t.Fatal(e)
			}
			if j.Operations[op].Path == manifestName {
				obstructLifecycleRollback(t, roots[0], ownerNameLockFile("old"))
				return errors.New("synthetic rollback obstruction")
			}
		}
		return nil
	}, makeOwnerRenamePlan)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, roots[0], ownerNameLockFile("old"), old)
	if !reflect.DeepEqual(peer, snapshot(t, roots[1])) {
		t.Fatal("peer restoration not attempted")
	}
	before := snapshot(t, roots[0])
	for _, check := range []bool{true, false} {
		_, e := WriteOwnerRenamesBatch(updates, check)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(before, snapshot(t, roots[0])) {
			t.Fatal("recovery evidence lost")
		}
	}
}

func TestOwnerRenameStagingFailureAfterOldMetadataBackup(t *testing.T) {
	root, _, u := renameFixture(t, true)
	before := snapshot(t, root)
	_, e := writeOwnersBatchWith(map[string]OwnerRenameUpdate{root: u}, false, func(_ int, phase string, op int, stage string) error {
		if phase == "apply" {
			var j journal
			if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
				t.Fatal(e)
			}
			if j.Operations[op].Path == manifestName {
				if e := os.Remove(filepath.Join(stage, *j.Operations[op].Staged)); e != nil {
					t.Fatal(e)
				}
			}
		}
		return nil
	}, makeOwnerRenamePlan)
	errorCode(t, e, "E_OUTPUT_COMMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("apply failure lost backed up old state")
	}
}
