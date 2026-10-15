package output

import (
	"encoding/json"
	"openapi-kt/core"
	"openapi-kt/internal/schema"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNameLockOwnersDedicatedJSONCannotEnterKotlinInventory(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, path := range []string{ownerNameLockFile("alpha"), "../Escaped.kt", ".private/Hidden.kt", "Model.json"} {
		root := t.TempDir()
		bad := a
		bad.Files = append(append([]core.GeneratedFile(nil), a.Files...), core.GeneratedFile{Path: path, Content: "{}"})
		u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": bad}, map[string][]byte{"alpha": nil}, false)
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwners(root, u, check)
			errorCode(t, e, "E_OUTPUT_PATH")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("unsafe generated inventory wrote")
		}
		hash := digest(a.NameLock)
		owners := map[string][]string{"alpha": {path}}
		locks := map[string]string{"alpha": hash}
		m := manifest{Version: 4, Files: map[string]string{path: hash}, Owners: &owners, NameLocks: &locks}
		b, _ := json.Marshal(m)
		put(t, filepath.Join(root, manifestName), b)
		_, e := ReadOwnerNameLock(root, "alpha")
		errorCode(t, e, "E_OUTPUT_PATH")
	}
}
func TestNameLockOwnersStateReadLimitAndManifestInventoryCap(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	root := t.TempDir()
	u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": a}, map[string][]byte{"alpha": nil}, false)
	if _, e := WriteNameLockedOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, ownerNameLockFile("alpha")), []byte(strings.Repeat(" ", schema.MaxNameLockBytes+1)))
	before := snapshot(t, root)
	_, e := ReadOwnerNameLock(root, "alpha")
	errorCode(t, e, "E_OUTPUT_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("oversized read mutated")
	}
	huge := a
	huge.NameLock = append(append([]byte(nil), a.NameLock...), []byte(strings.Repeat(" ", schema.MaxNameLockBytes+1-len(a.NameLock)))...)
	absent := filepath.Join(t.TempDir(), "missing")
	u.Generated["alpha"] = huge
	_, e = WriteNameLockedOwners(absent, u, false)
	errorCode(t, e, "E_NAME_LOCK_LIMIT")
	if _, e := os.Stat(absent); !os.IsNotExist(e) {
		t.Fatal("incoming oversized state created root")
	}
	// The strict decoder enforces the inventory cap before payload IO, not
	// only the generated-owner input validator.
	m := manifest{Version: 4, Files: map[string]string{}, Owners: &map[string][]string{}, NameLocks: &map[string]string{}}
	for i := 0; i <= maxLockedOwners; i++ {
		owner := "owner" + strings.Repeat("x", i)
		(*m.Owners)[owner] = []string{"A.kt"}
		(*m.NameLocks)[owner] = digest(a.NameLock)
	}
	m.Files["A.kt"] = digest([]byte("synthetic"))
	b, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), b)
	_, e = ReadOwnerNameLock(root, "alpha")
	errorCode(t, e, "E_NAME_LOCK_LIMIT")
}
func TestNameLockOwnersBatchFailureAfterStateBackupMoves(t *testing.T) {
	old, next := ownerBatchFixture(t)
	if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
		t.Fatal(e)
	}
	roots := sortedKeys(old)
	before := map[string]map[string]snapshotEntry{}
	for _, r := range roots {
		before[r] = snapshot(t, r)
	}
	seen := false
	_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
		if ri == 1 && phase == "apply" {
			var j journal
			_ = json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j)
			op := j.Operations[index]
			if op.Path == ownerNameLockFile("beta") {
				seen = true
				return os.Remove(filepath.Join(stage, *op.Staged))
			}
		}
		return nil
	}, makeNameLockedOwnersPlan)
	errorCode(t, e, "E_OUTPUT_COMMIT")
	if !seen {
		t.Fatal("metadata never entered journal")
	}
	for _, r := range roots {
		if !reflect.DeepEqual(before[r], snapshot(t, r)) {
			t.Fatal("state backup-move failure lost old output")
		}
	}
}
