package output

import (
	"bytes"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func lifecycleArtifacts(t testing.TB, req core.GenerationRequest, action NameLockAction) core.GenerationArtifacts {
	t.Helper()
	req.PreviousNameLock = nil
	req.Options.LockHoistedNames = action == NameLockReset
	return singleGenerated(t, req)
}

func lifecycleOwnerUpdate(owner string, a core.GenerationArtifacts, before []byte, action NameLockAction) NameLockedOwnerUpdate {
	u := lockedUpdate([]string{owner}, map[string]core.GenerationArtifacts{owner: a}, map[string][]byte{owner: before}, false)
	u.NameLockActions = map[string]NameLockAction{owner: action}
	return u
}

func requireLifecycleFileUnchanged(t testing.TB, before, after map[string]snapshotEntry, path string) {
	t.Helper()
	key := filepath.FromSlash(path)
	old, hadOld := before[key]
	current, exists := after[key]
	if !hadOld || !exists || !reflect.DeepEqual(old, current) {
		t.Fatal("lifecycle changed or omitted an unchanged owned file", path)
	}
}

func TestNameLockLifecycleSingleMetadataOnlyNoopCheckAndDisable(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		root := t.TempDir()
		req := singleLockRequest(t, style, true)
		old := singleGenerated(t, req)
		if _, e := WriteNameLocked(root, old, nil, false); e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten"))
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			if changed, e := WriteNameLockLifecycle(root, old, old.NameLock, NameLockReset, check); e != nil || changed {
				t.Fatal("reset of identical state is a noop", changed, e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("noop reset changed bytes/modes/mtimes")
			}
		}
		// Equivalent, valid JSON bytes force a metadata-only transaction.
		next := old
		next.NameLock = append(append([]byte(nil), old.NameLock...), '\n')
		if changed, e := WriteNameLockLifecycle(root, next, old.NameLock, NameLockReset, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("reset check wrote")
		}
		p, e := makeNameLockPlan(root, next, old.NameLock, NameLockReset)
		if e != nil || len(p.Writes) != 1 || len(p.Removals) != 0 || p.Manifest.Version != 3 {
			t.Fatal("reset must plan only metadata and manifest", p, e)
		}
		if changed, e := WriteNameLockLifecycle(root, next, old.NameLock, NameLockReset, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		afterReset := snapshot(t, root)
		for _, f := range old.Files {
			requireLifecycleFileUnchanged(t, before, afterReset, f.Path)
		}
		off := lifecycleArtifacts(t, req, NameLockDisable)
		if !reflect.DeepEqual(old.Files, off.Files) {
			t.Fatal("fixture should have identical locked/unlocked Kotlin")
		}
		for _, check := range []bool{true, false} {
			_, e := Write(root, off, check)
			errorCode(t, e, "E_MANIFEST_VERSION")
		}
		p, e = makeNameLockPlan(root, off, next.NameLock, NameLockDisable)
		if e != nil || len(p.Writes) != 0 || !reflect.DeepEqual(p.Removals, []string{nameLockFile}) || p.Manifest.Version != 1 || p.Manifest.NameLock != nil {
			t.Fatal("disable must explicitly remove metadata and plan v1", p, e)
		}
		if changed, e := WriteNameLockLifecycle(root, off, next.NameLock, NameLockDisable, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		if !reflect.DeepEqual(afterReset, snapshot(t, root)) {
			t.Fatal("disable check wrote")
		}
		if changed, e := WriteNameLockLifecycle(root, off, next.NameLock, NameLockDisable, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		after := snapshot(t, root)
		if _, exists := after[nameLockFile]; exists {
			t.Fatal("disable retained state")
		}
		for _, f := range old.Files {
			requireLifecycleFileUnchanged(t, before, after, f.Path)
		}
		if !reflect.DeepEqual(before["Handwritten.kt"], after["Handwritten.kt"]) {
			t.Fatal("lifecycle touched handwritten file")
		}
		for _, check := range []bool{true, false} {
			if changed, e := Write(root, off, check); e != nil || changed {
				t.Fatal("ordinary v1 noop after explicit disable", changed, e)
			}
			_, e := WriteNameLockLifecycle(root, off, nil, NameLockDisable, check)
			errorCode(t, e, "E_NAME_LOCK_LIFECYCLE")
			if !reflect.DeepEqual(after, snapshot(t, root)) {
				t.Fatal("post-disable noop/rejection mutated output")
			}
		}
	}
}

func TestNameLockLifecycleResetAllowsRemovedOccurrenceWithSameProfile(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, addedSinglePeer(t, req))
	if _, e := WriteNameLocked(root, old, nil, false); e != nil {
		t.Fatal(e)
	}
	req.PreviousNameLock = old.NameLock
	_, generationError := core.Generate(req)
	diagnostics, ok := generationError.(core.Diagnostics)
	if !ok || len(diagnostics) != 1 || diagnostics[0].Code != "E_NAME_LOCK_REMOVED" {
		t.Fatal("fixture must require reset", generationError)
	}
	next := lifecycleArtifacts(t, req, NameLockReset)
	if changed, e := WriteNameLockLifecycle(root, next, old.NameLock, NameLockReset, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	current, e := ReadNameLock(root)
	if e != nil || !bytes.Equal(current, next.NameLock) {
		t.Fatal("reset did not commit fresh state", e)
	}
	req.PreviousNameLock = current
	if !reflect.DeepEqual(singleGenerated(t, req), next) {
		t.Fatal("reset state does not replay deterministically")
	}
}

func TestNameLockLifecycleOwnersMixedActionsRetainLockedPeer(t *testing.T) {
	root := t.TempDir()
	ar, br, gr := ownerLockRequest(t, "alpha", core.DataClass, true), ownerLockRequest(t, "beta", core.DataClass, true), ownerLockRequest(t, "gamma", core.DataClass, true)
	a, b, g := ownedGenerated(t, ar), ownedGenerated(t, br), ownedGenerated(t, gr)
	initial := lockedUpdate([]string{"alpha", "beta", "gamma"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b, "gamma": g}, map[string][]byte{"alpha": nil, "beta": nil, "gamma": nil}, true)
	if _, e := WriteNameLockedOwners(root, initial, false); e != nil {
		t.Fatal(e)
	}
	an := ownedGenerated(t, addedSinglePeer(t, ar))
	br.Options.LockHoistedNames = false
	bn := ownedGenerated(t, br)
	u := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": an, "beta": bn}, map[string][]byte{"alpha": a.NameLock, "beta": b.NameLock}, false)
	u.NameLockActions = map[string]NameLockAction{"alpha": NameLockReset, "beta": NameLockDisable}
	before := snapshot(t, root)
	if changed, e := WriteNameLockedOwners(root, u, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("mixed lifecycle check wrote")
	}
	if changed, e := WriteNameLockedOwners(root, u, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	m, e := readManifestKind(root, true, true)
	if e != nil || m.Version != 4 || len(*m.Owners) != 3 || len(*m.NameLocks) != 2 || (*m.NameLocks)["alpha"] != digest(an.NameLock) || (*m.NameLocks)["gamma"] != digest(g.NameLock) {
		t.Fatal("mixed actions lost state/owner inventory", m, e)
	}
	after := snapshot(t, root)
	if _, exists := after[ownerNameLockFile("beta")]; exists {
		t.Fatal("mixed actions retained disabled state")
	}
	for _, f := range append(b.Files, g.Files...) {
		requireLifecycleFileUnchanged(t, before, after, f.Path)
	}
	if !reflect.DeepEqual(before[ownerNameLockFile("gamma")], after[ownerNameLockFile("gamma")]) {
		t.Fatal("mixed lifecycle changed retained state")
	}
}

func TestNameLockLifecycleOwnersPartialSharedMetadataAndLastLock(t *testing.T) {
	root := t.TempDir()
	ar, br := ownerLockRequest(t, "alpha", core.DataClass, true), ownerLockRequest(t, "beta", core.DataClass, true)
	a, b := ownedGenerated(t, ar), ownedGenerated(t, br)
	u := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
	if _, e := WriteNameLockedOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	foreign := ownerNameLockFile("foreign")
	put(t, filepath.Join(root, foreign), []byte("synthetic unowned state"))
	put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten"))
	before := snapshot(t, root)
	reset := lifecycleOwnerUpdate("alpha", a, a.NameLock, NameLockReset)
	for _, check := range []bool{true, false} {
		if changed, e := WriteNameLockedOwners(root, reset, check); e != nil || changed {
			t.Fatal("named identical reset noop", changed, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("named reset noop mutated output")
		}
	}
	next := a
	next.NameLock = append(append([]byte(nil), a.NameLock...), '\n')
	reset.Generated["alpha"] = next
	if changed, e := WriteNameLockedOwners(root, reset, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("named reset check wrote")
	}
	if _, e := WriteNameLockedOwners(root, reset, false); e != nil {
		t.Fatal(e)
	}
	for _, owner := range []string{"alpha", "beta"} {
		oldState, req := next.NameLock, ar
		if owner == "beta" {
			oldState, req = b.NameLock, br
		}
		req.Options.LockHoistedNames = false
		off := ownedGenerated(t, req)
		disable := lifecycleOwnerUpdate(owner, off, oldState, NameLockDisable)
		stable := snapshot(t, root)
		if changed, e := WriteNameLockedOwners(root, disable, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		if !reflect.DeepEqual(stable, snapshot(t, root)) {
			t.Fatal("named disable check wrote")
		}
		if _, e := WriteNameLockedOwners(root, disable, false); e != nil {
			t.Fatal(e)
		}
		m, e := readManifestKind(root, true, true)
		if e != nil || m.Version != 4 || len(*m.Owners) != 2 || len((*m.NameLocks)[owner]) != 0 {
			t.Fatal("disable lost owner or downgraded v4", m, e)
		}
		if owner == "alpha" && (*m.NameLocks)["beta"] != digest(b.NameLock) || owner == "beta" && len(*m.NameLocks) != 0 {
			t.Fatal("incorrect retained/last lock inventory")
		}
		after := snapshot(t, root)
		if _, exists := after[ownerNameLockFile(owner)]; exists {
			t.Fatal("disabled metadata retained")
		}
		for _, f := range append(a.Files, b.Files...) {
			requireLifecycleFileUnchanged(t, before, after, f.Path)
		}
		if owner == "alpha" && !reflect.DeepEqual(before[ownerNameLockFile("beta")], after[ownerNameLockFile("beta")]) {
			t.Fatal("partial disable rewrote retained state")
		}
		if !reflect.DeepEqual(before[foreign], after[foreign]) || !reflect.DeepEqual(before["Handwritten.kt"], after["Handwritten.kt"]) {
			t.Fatal("lifecycle touched unowned output")
		}
		plain := lockedUpdate([]string{owner}, map[string]core.GenerationArtifacts{owner: off}, nil, false)
		for _, check := range []bool{true, false} {
			if changed, e := WriteNameLockedOwners(root, plain, check); e != nil || changed {
				t.Fatal("unlocked owner noop must keep v4", changed, e)
			}
		}
	}
	if _, e := os.Stat(filepath.Join(root, "shared", "Marker.kt")); e != nil {
		t.Fatal("last lock removal lost shared Kotlin", e)
	}
	if _, e := WriteNameLockedOwners(root, lockedUpdate(nil, nil, nil, true), false); e != nil {
		t.Fatal(e)
	}
	m, e := readManifestKind(root, true, true)
	if e != nil || m.Version != 4 || len(m.Files) != 0 || len(*m.Owners) != 0 || len(*m.NameLocks) != 0 {
		t.Fatal("empty reconcile downgraded v4", m, e)
	}
	for _, f := range append(a.Files, b.Files...) {
		if _, e := os.Stat(filepath.Join(root, f.Path)); !os.IsNotExist(e) {
			t.Fatal("empty reconcile retained Kotlin", f.Path, e)
		}
	}
	if !reflect.DeepEqual(before[foreign], snapshot(t, root)[foreign]) {
		t.Fatal("reconcile deleted unowned metadata")
	}
}
