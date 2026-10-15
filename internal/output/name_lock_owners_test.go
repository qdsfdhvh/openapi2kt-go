package output

import (
	"encoding/json"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNameLockOwnersDefaultEnvelopeControl(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		root := t.TempDir()
		a := singleGenerated(t, singleLockRequest(t, style, false))
		u := OwnerUpdate{ActiveOwners: []string{"alpha", "beta"}, Generated: map[string]core.GenerationArtifacts{"alpha": a, "beta": a}, Reconcile: true}
		if _, e := WriteOwners(root, u, false); e != nil {
			t.Fatal(e)
		}
		var m manifest
		if e := json.Unmarshal(mustRead(t, filepath.Join(root, manifestName)), &m); e != nil {
			t.Fatal(e)
		}
		if m.Version != 2 || len(m.Files) != len(a.Files) || len(*m.Owners) != 2 || !reflect.DeepEqual((*m.Owners)["alpha"], (*m.Owners)["beta"]) {
			t.Fatal("default named envelope")
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			changed, e := WriteOwners(root, u, check)
			if e != nil || changed {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("default named noop mutated")
			}
		}
		for _, check := range []bool{true, false} {
			if changed, e := WriteNameLockedOwners(root, NameLockedOwnerUpdate{OwnerUpdate: u}, check); e != nil || changed {
				t.Fatal("explicit all-off must retain version2 envelope", e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("all-off explicit writer changed v2")
			}
		}
		u.Generated = map[string]core.GenerationArtifacts{"alpha": a}
		u.Reconcile = false
		if changed, e := WriteOwners(root, u, false); e != nil || changed {
			t.Fatal(e)
		}
		u.ActiveOwners = []string{"alpha"}
		u.Reconcile = true
		if changed, e := WriteOwners(root, u, false); e != nil || !changed {
			t.Fatal(e)
		}
		after := snapshot(t, root)
		for _, f := range a.Files {
			if !reflect.DeepEqual(before[f.Path], after[f.Path]) {
				t.Fatal("shared last peer removal rewrote Kotlin")
			}
		}
	}
}
func TestNameLockOwnersDesiredSuccess(t *testing.T) {
	root := t.TempDir()
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	u := NameLockedOwnerUpdate{OwnerUpdate: OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": a}}, PreviousNameLocks: map[string][]byte{"alpha": nil}}
	if changed, e := WriteNameLockedOwners(root, u, false); e != nil || !changed {
		t.Fatal("desired named-owner managed lock success", e)
	}
}
