package output

import (
	"bytes"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func ownerLockRequest(t testing.TB, owner string, style core.ModelStyle, lock bool) core.GenerationRequest {
	r := singleLockRequest(t, style, lock)
	r.Options.PackageName = "org.example.owner" + owner
	return r
}
func ownedGenerated(t testing.TB, r core.GenerationRequest) core.GenerationArtifacts {
	a := singleGenerated(t, r)
	a.Files = append(a.Files, core.GeneratedFile{Path: "shared/Marker.kt", Content: "// synthetic shared adapter membership sentinel"})
	return a
}
func lockedUpdate(active []string, generated map[string]core.GenerationArtifacts, previous map[string][]byte, reconcile bool) NameLockedOwnerUpdate {
	return NameLockedOwnerUpdate{OwnerUpdate: OwnerUpdate{ActiveOwners: active, Generated: generated, Reconcile: reconcile}, PreviousNameLocks: previous}
}
func TestNameLockOwnersMixedPartialReplayReconcile(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		root := filepath.Join(t.TempDir(), "new")
		req := ownerLockRequest(t, "alpha", style, true)
		a := ownedGenerated(t, req)
		b := ownedGenerated(t, ownerLockRequest(t, "beta", style, true))
		g := ownedGenerated(t, ownerLockRequest(t, "gamma", style, false))
		u := lockedUpdate([]string{"alpha", "beta", "gamma"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b, "gamma": g}, map[string][]byte{"alpha": nil, "beta": nil}, true)
		if state, e := ReadOwnerNameLock(root, "alpha"); e != nil || state != nil {
			t.Fatal(e)
		}
		if changed, e := WriteNameLockedOwners(root, u, true); e != nil || !changed {
			t.Fatal(e)
		}
		if _, e := os.Stat(root); !os.IsNotExist(e) {
			t.Fatal("check created root")
		}
		if _, e := WriteNameLockedOwners(root, u, false); e != nil {
			t.Fatal(e)
		}
		m, e := readManifestKind(root, true, true)
		if e != nil || m.Version != 4 || len(*m.NameLocks) != 2 || len(*m.Owners) != 3 {
			t.Fatal(m, e)
		}
		u.PreviousNameLocks = map[string][]byte{"alpha": a.NameLock, "beta": b.NameLock}
		before := snapshot(t, root)
		for _, artifact := range u.Generated {
			for _, f := range artifact.Files {
				if !bytes.Equal(mustRead(t, filepath.Join(root, f.Path)), []byte(f.Content)) {
					t.Fatal("initial committed Kotlin differs from actual core artifact", f.Path)
				}
			}
		}
		for _, check := range []bool{true, false} {
			if changed, e := WriteNameLockedOwners(root, u, check); e != nil || changed {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("mixed noop changed files")
			}
		}
		state, e := ReadOwnerNameLock(root, "alpha")
		if e != nil || !bytes.Equal(state, a.NameLock) {
			t.Fatal(e)
		}
		req = addedSinglePeer(t, req)
		req.PreviousNameLock = state
		next := ownedGenerated(t, req)
		partial := lockedUpdate([]string{"alpha", "gamma"}, map[string]core.GenerationArtifacts{"alpha": next}, map[string][]byte{"alpha": state}, false)
		if _, e := WriteNameLockedOwners(root, partial, false); e != nil {
			t.Fatal(e)
		}
		after := snapshot(t, root)
		for _, f := range next.Files {
			if !bytes.Equal(mustRead(t, filepath.Join(root, f.Path)), []byte(f.Content)) {
				t.Fatal("grown committed Kotlin differs from actual core artifact", f.Path)
			}
		}
		for _, f := range append(b.Files, g.Files...) {
			if !reflect.DeepEqual(before[f.Path], after[f.Path]) {
				t.Fatal("partial changed peer", f.Path)
			}
		}
		if !reflect.DeepEqual(before[ownerNameLockFile("beta")], after[ownerNameLockFile("beta")]) {
			t.Fatal("partial rewrote peer state")
		}
		current, e := ReadOwnerNameLock(root, "alpha")
		if e != nil {
			t.Fatal(e)
		}
		req.PreviousNameLock = current
		if !reflect.DeepEqual(ownedGenerated(t, req), next) {
			t.Fatal("read state replay differs")
		}
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwners(root, partial, check)
			errorCode(t, e, "E_NAME_LOCK_STALE")
			if !reflect.DeepEqual(after, snapshot(t, root)) {
				t.Fatal("stale partial mutated")
			}
		}
		offReq := req
		offReq.Options.LockHoistedNames = false
		offReq.PreviousNameLock = nil
		off := ownedGenerated(t, offReq)
		disabled := lockedUpdate(u.ActiveOwners, map[string]core.GenerationArtifacts{"alpha": off}, nil, false)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwners(root, disabled, check)
			errorCode(t, e, "E_NAME_LOCK_DISABLED")
		}
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("implicit disable changed files")
		}
		// A reader does not load unselected peer payloads; a writer validates all
		// owned states, even those being explicitly removed by reconciliation.
		put(t, filepath.Join(root, ownerNameLockFile("beta")), []byte("synthetic edited state"))
		edited := snapshot(t, root)
		if _, e := ReadOwnerNameLock(root, "alpha"); e != nil {
			t.Fatal(e)
		}
		reconcile := lockedUpdate([]string{"alpha", "gamma"}, nil, nil, true)
		_, e = WriteNameLockedOwners(root, reconcile, false)
		errorCode(t, e, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(edited, snapshot(t, root)) {
			t.Fatal("reconcile deleted edited state")
		}
		put(t, filepath.Join(root, ownerNameLockFile("beta")), b.NameLock)
		if _, e := WriteNameLockedOwners(root, reconcile, false); e != nil {
			t.Fatal(e)
		}
		if _, e := os.Stat(filepath.Join(root, ownerNameLockFile("beta"))); !os.IsNotExist(e) {
			t.Fatal("removed owner state retained")
		}
		foreign := ownerNameLockFile("foreign")
		put(t, filepath.Join(root, foreign), []byte("synthetic unowned metadata"))
		foreignBefore := snapshot(t, root)[foreign]
		reconcile.ActiveOwners = []string{"gamma"}
		if _, e := WriteNameLockedOwners(root, reconcile, false); e != nil {
			t.Fatal(e)
		}
		m, e = readManifestKind(root, true, true)
		if e != nil || m.Version != 4 || len(*m.NameLocks) != 0 {
			t.Fatal("implicit downgrade", e)
		}
		if !reflect.DeepEqual(foreignBefore, snapshot(t, root)[foreign]) {
			t.Fatal("deleted unowned metadata")
		}
		for _, check := range []bool{true, false} {
			_, e := WriteOwners(root, OwnerUpdate{ActiveOwners: []string{"gamma"}, Generated: map[string]core.GenerationArtifacts{"gamma": g}}, check)
			errorCode(t, e, "E_MANIFEST_VERSION")
		}
	}
}
func TestNameLockOwnersVersionTwoAndLegacyMetadataOnlyMigration(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		root := t.TempDir()
		req := ownerLockRequest(t, "alpha", core.DataClass, false)
		off := singleGenerated(t, req)
		u := OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": off}}
		if legacy {
			if _, e := Write(root, off, false); e != nil {
				t.Fatal(e)
			}
		} else {
			if _, e := WriteOwners(root, u, false); e != nil {
				t.Fatal(e)
			}
		}
		before := snapshot(t, root)
		state, e := ReadOwnerNameLock(root, "alpha")
		if e != nil || state != nil {
			t.Fatal(e)
		}
		req.Options.LockHoistedNames = true
		a := singleGenerated(t, req)
		if !reflect.DeepEqual(a.Files, off.Files) {
			t.Fatal("opt-in altered Kotlin")
		}
		update := lockedUpdate(u.ActiveOwners, map[string]core.GenerationArtifacts{"alpha": a}, map[string][]byte{"alpha": nil}, false)
		if legacy {
			_, e := WriteNameLockedOwners(root, update, true)
			errorCode(t, e, "E_MANIFEST_OWNER_MIGRATION")
			update.AdoptLegacy = "alpha"
		}
		if changed, e := WriteNameLockedOwners(root, update, true); e != nil || !changed {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("migration check wrote")
		}
		if _, e := WriteNameLockedOwners(root, update, false); e != nil {
			t.Fatal(e)
		}
		after := snapshot(t, root)
		for _, f := range a.Files {
			if !reflect.DeepEqual(before[f.Path], after[f.Path]) {
				t.Fatal("metadata-only migration rewrote Kotlin")
			}
		}
		if !bytes.Equal(mustRead(t, filepath.Join(root, ownerNameLockFile("alpha"))), a.NameLock) {
			t.Fatal("wrong migrated state")
		}
	}
}
