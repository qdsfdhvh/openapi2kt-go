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

func TestSingleNameLockEveryCommitBoundaryRestoresStateAndKotlin(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	req = addedSinglePeer(t, req)
	req.PreviousNameLock = old.NameLock
	next := singleGenerated(t, req)
	// Include stale removal and a newly created parent: these are ordinary
	// output adapter inputs, not replacements for the core-generated models.
	old.Files = append(old.Files, core.GeneratedFile{Path: "stale/Synthetic.kt", Content: "// synthetic old owned sentinel"})
	next.Files = append(next.Files, core.GeneratedFile{Path: "new/deep/Synthetic.kt", Content: "// synthetic new owned sentinel"})
	var operations int
	for index := 0; index < 100; index++ {
		root := t.TempDir()
		if _, e := WriteNameLocked(root, old, nil, false); e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(root, "Handwritten.kt"), []byte("synthetic handwritten"))
		before := snapshot(t, root)
		p, e := makeNameLockedPlan(root, next, old.NameLock)
		if e != nil {
			t.Fatal(e)
		}
		operations = len(p.Writes) + len(p.Removals) + 1
		if index >= operations {
			break
		}
		e = commit(root, p, func(op int, stage string) error {
			var j journal
			if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
				t.Fatal(e)
			}
			if j.Operations[len(j.Operations)-1].Path != manifestName {
				t.Fatal("manifest not last")
			}
			found := false
			for _, entry := range j.Operations {
				if entry.Path == nameLockFile {
					found = true
					if !entry.HadOld {
						t.Fatal("metadata missing old backup")
					}
				}
			}
			if !found {
				t.Fatal("metadata outside journal")
			}
			if op == index {
				return errors.New("synthetic boundary")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("failure lost bytes/modes/mtimes/state", index)
		}
	}
	if operations < 4 {
		t.Fatal("insufficient boundaries")
	}
	t.Log("injected every", operations, "Kotlin/state/stale-removal/manifest boundary")
}
func TestSingleNameLockFailureAfterMetadataBackup(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	if _, e := WriteNameLocked(root, old, nil, false); e != nil {
		t.Fatal(e)
	}
	req = addedSinglePeer(t, req)
	req.PreviousNameLock = old.NameLock
	next := singleGenerated(t, req)
	before := snapshot(t, root)
	p, e := makeNameLockedPlan(root, next, old.NameLock)
	if e != nil {
		t.Fatal(e)
	}
	e = commit(root, p, func(index int, stage string) error {
		var j journal
		_ = json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j)
		if j.Operations[index].Path == nameLockFile {
			return os.Remove(filepath.Join(stage, *j.Operations[index].Staged))
		}
		return nil
	})
	errorCode(t, e, "E_OUTPUT_COMMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("metadata backup-move rollback changed tree")
	}
}
func TestSingleNameLockRollbackObstructionRetainsEvidence(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	if _, e := WriteNameLocked(root, old, nil, false); e != nil {
		t.Fatal(e)
	}
	req = addedSinglePeer(t, req)
	req.PreviousNameLock = old.NameLock
	next := singleGenerated(t, req)
	p, e := makeNameLockedPlan(root, next, old.NameLock)
	if e != nil {
		t.Fatal(e)
	}
	e = commit(root, p, func(index int, stage string) error {
		if index == 1 {
			if e := os.Remove(filepath.Join(root, nameLockFile)); e != nil {
				t.Fatal(e)
			}
			if e := os.Mkdir(filepath.Join(root, nameLockFile), 0700); e != nil {
				t.Fatal(e)
			}
			return errors.New("synthetic obstructed rollback")
		}
		return nil
	})
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	stages, e2 := filepath.Glob(filepath.Join(root, stagePrefix+"*"))
	if e2 != nil || len(stages) != 1 {
		t.Fatal(stages, e2)
	}
	var j journal
	_ = json.Unmarshal(mustRead(t, filepath.Join(stages[0], "transaction.json")), &j)
	for i, entry := range j.Operations {
		if entry.Path == nameLockFile {
			if string(mustRead(t, filepath.Join(stages[0], entry.Backup))) != string(old.NameLock) {
				t.Fatal("lost name-lock backup")
			}
			_ = i
		}
	}
	_, e = ReadNameLock(root)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	for _, check := range []bool{false, true} {
		_, e = WriteNameLocked(root, next, old.NameLock, check)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
	}
}
