package output

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestOwnerRenamePreservesNoncanonicalRawStateAndMixedVersion4(t *testing.T) {
	root, a, u := renameFixture(t, true)
	// Valid noncanonical old state is preserved as raw bytes; the generated
	// candidate may canonicalize it without changing allocations or occurrences.
	old := append(append([]byte(nil), a.NameLock...), '\n')
	put(t, filepath.Join(root, ownerNameLockFile("old")), old)
	m, e := readManifestKind(root, true, true)
	if e != nil {
		t.Fatal(e)
	}
	(*m.NameLocks)["old"] = digest(old)
	data, e2 := json.Marshal(m)
	if e2 != nil {
		t.Fatal(e2)
	}
	put(t, filepath.Join(root, manifestName), data)
	state, e := ReadOwnerRenameState(root)
	if e != nil {
		t.Fatal(e)
	}
	u.Previous = state
	if _, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil {
		t.Fatal(e)
	}
	if string(mustRead(t, filepath.Join(root, ownerNameLockFile("new")))) != string(old) {
		t.Fatal("raw state canonicalized")
	}

	root, _, u = renameFixture(t, false)
	locked := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	if _, e := WriteNameLockedOwners(root, lockedUpdate([]string{"peer"}, map[string]core.GenerationArtifacts{"peer": locked}, map[string][]byte{"peer": nil}, false), false); e != nil {
		t.Fatal(e)
	}
	state, e = ReadOwnerRenameState(root)
	if e != nil {
		t.Fatal(e)
	}
	u.Previous = state
	before := snapshot(t, root)
	if _, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil {
		t.Fatal(e)
	}
	after := snapshot(t, root)
	if !reflect.DeepEqual(before[ownerNameLockFile("peer")], after[ownerNameLockFile("peer")]) {
		t.Fatal("locked peer changed during unlocked rename")
	}
	m, e = readManifestKind(root, true, true)
	if e != nil || m.Version != 4 || len(*m.NameLocks) != 1 || (*m.NameLocks)["new"] != "" {
		t.Fatal("mixed v4 interpretation changed", m, e)
	}
}
