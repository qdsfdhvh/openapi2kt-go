package output

import (
	"bytes"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNameLockOwnersRetainedPeerReloadUnderBatchLocks(t *testing.T) {
	root := t.TempDir()
	ar := ownerLockRequest(t, "alpha", core.DataClass, true)
	br := ownerLockRequest(t, "beta", core.DataClass, true)
	a, b := ownedGenerated(t, ar), ownedGenerated(t, br)
	initial := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
	if _, e := WriteNameLockedOwners(root, initial, false); e != nil {
		t.Fatal(e)
	}
	ar = addedSinglePeer(t, ar)
	ar.PreviousNameLock = a.NameLock
	an := ownedGenerated(t, ar)
	br = addedSinglePeer(t, br)
	br.PreviousNameLock = b.NameLock
	bn := ownedGenerated(t, br)
	partial := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": an}, map[string][]byte{"alpha": a.NameLock}, false)
	var competitor map[string]snapshotEntry
	_, e := writeOwnersBatchWith(map[string]NameLockedOwnerUpdate{root: partial}, false, func(ri int, phase string, op int, stage string) error {
		if phase == "prelock" {
			peer := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"beta": bn}, map[string][]byte{"beta": b.NameLock}, false)
			if _, e := WriteNameLockedOwners(root, peer, false); e != nil {
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
		t.Fatal("lost peer's newly committed state", e)
	}
	after := snapshot(t, root)
	if !reflect.DeepEqual(competitor[ownerNameLockFile("beta")], after[ownerNameLockFile("beta")]) {
		t.Fatal("rewrote retained peer state")
	}
	for _, f := range bn.Files {
		if !reflect.DeepEqual(competitor[f.Path], after[f.Path]) {
			t.Fatal("lost current retained peer file", filepath.Base(f.Path))
		}
	}
}
