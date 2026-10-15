package output

import (
	"bytes"
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNameLockLifecycleRejectsMissingOldSelectedKotlinBeforePlanning(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	for _, named := range []bool{false, true} {
		for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
			for _, missingKind := range []string{"kept", "removed"} {
				t.Run(map[bool]string{false: "direct/", true: "named/"}[named]+string(action)+"/"+missingKind, func(t *testing.T) {
					root := t.TempDir()
					old := singleGenerated(t, req)
					old.Files = append(old.Files, core.GeneratedFile{Path: "stale/Selected.kt", Content: "// synthetic old selected Kotlin"})
					seedLifecycleTest(t, root, named, old)
					missingPath := old.Files[0].Path
					if missingKind == "removed" {
						missingPath = "stale/Selected.kt"
					}
					if e := os.Remove(filepath.Join(root, filepath.FromSlash(missingPath))); e != nil {
						t.Fatal(e)
					}
					before := snapshot(t, root)
					next := lifecycleArtifacts(t, req, action)
					for _, check := range []bool{true, false} {
						_, e := writeLifecycleTest(root, named, next, old.NameLock, action, check)
						errorCode(t, e, "E_OUTPUT_MISSING")
						if !reflect.DeepEqual(before, snapshot(t, root)) {
							t.Fatal("lifecycle repaired or removed missing old selected Kotlin")
						}
					}
				})
			}
		}
	}
}

func TestNameLockLifecycleSingleRechecksOldSelectedKotlinUnderLock(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		for _, damage := range []string{"missing", "modified"} {
			root := t.TempDir()
			req := singleLockRequest(t, core.DataClass, true)
			old := singleGenerated(t, req)
			seedLifecycleTest(t, root, false, old)
			next := lifecycleArtifacts(t, addedSinglePeer(t, req), action)
			var damaged map[string]snapshotEntry
			calls := 0
			_, e := writePlanned(root, false, func(root string) (*plan, *diagnostic.Diagnostic) {
				calls++
				p, e := makeNameLockPlan(root, next, old.NameLock, action)
				if e == nil && calls == 1 {
					path := filepath.Join(root, filepath.FromSlash(old.Files[0].Path))
					if damage == "missing" {
						if e := os.Remove(path); e != nil {
							t.Fatal(e)
						}
					} else {
						put(t, path, []byte("// synthetic Kotlin edit after preflight"))
					}
					damaged = snapshot(t, root)
				}
				return p, e
			})
			code := "E_OUTPUT_MISSING"
			if damage == "modified" {
				code = "E_OUTPUT_MODIFIED"
			}
			errorCode(t, e, code)
			if calls != 2 || !reflect.DeepEqual(damaged, snapshot(t, root)) {
				t.Fatal("locked lifecycle repaired or overwrote changed selected Kotlin", action, damage, calls)
			}
			current, e := ReadNameLock(root)
			if e != nil || !bytes.Equal(current, old.NameLock) {
				t.Fatal("selected Kotlin damage changed stored state", e)
			}
		}
	}
}

func TestNameLockLifecycleBatchRechecksMissingOldSelectedKotlinBeforeAnyStage(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		old, next := lifecycleBatchFixture(t, action)
		if _, e := WriteNameLockedOwnersBatch(old, false); e != nil {
			t.Fatal(e)
		}
		roots := sortedKeys(old)
		beforeFirst := snapshot(t, roots[0])
		var damaged map[string]snapshotEntry
		staged := false
		_, e := writeOwnersBatchWith(next, false, func(ri int, phase string, index int, stage string) error {
			if phase == "prelock" {
				path := filepath.Join(roots[1], filepath.FromSlash(old[roots[1]].Generated["beta"].Files[0].Path))
				if e := os.Remove(path); e != nil {
					t.Fatal(e)
				}
				damaged = snapshot(t, roots[1])
			}
			if phase == "prepare" {
				staged = true
			}
			return nil
		}, makeNameLockedOwnersPlan)
		errorCode(t, e, "E_OUTPUT_MISSING")
		if staged || !reflect.DeepEqual(beforeFirst, snapshot(t, roots[0])) || !reflect.DeepEqual(damaged, snapshot(t, roots[1])) {
			t.Fatal("batch lifecycle staged early or repaired missing selected Kotlin", action)
		}
	}
}

func TestNameLockLifecycleOwnersReconcileRequiresOldSelectedSharedKotlin(t *testing.T) {
	for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
		root := t.TempDir()
		ar, br := ownerLockRequest(t, "alpha", core.DataClass, true), ownerLockRequest(t, "beta", core.DataClass, true)
		a, b := ownedGenerated(t, ar), ownedGenerated(t, br)
		initial := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
		if _, e := WriteNameLockedOwners(root, initial, false); e != nil {
			t.Fatal(e)
		}
		ar.Options.LockHoistedNames = action == NameLockReset
		next := ownedGenerated(t, ar)
		u := lifecycleOwnerUpdate("alpha", next, a.NameLock, action)
		u.Reconcile = true
		if e := os.Remove(filepath.Join(root, "shared", "Marker.kt")); e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwners(root, u, check)
			errorCode(t, e, "E_OUTPUT_MISSING")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("lifecycle+reconcile repaired missing selected shared Kotlin")
			}
		}
	}
}

func TestNameLockLifecycleOldFilesGuardKeepsCASOnStateBytes(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
			root := t.TempDir()
			req := singleLockRequest(t, core.DataClass, true)
			old := singleGenerated(t, req)
			seedLifecycleTest(t, root, named, old)
			next := lifecycleArtifacts(t, req, action)
			competing := old
			competing.Files = append([]core.GeneratedFile(nil), old.Files...)
			competing.Files[0].Content += "\n// synthetic cooperative update with unchanged lock state\n"
			compete := func() {
				t.Helper()
				var e *diagnostic.Diagnostic
				if named {
					u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": competing}, map[string][]byte{"alpha": old.NameLock}, false)
					_, e = WriteNameLockedOwners(root, u, false)
				} else {
					_, e = WriteNameLocked(root, competing, old.NameLock, false)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
			var changed bool
			var e *diagnostic.Diagnostic
			if named {
				u := lifecycleOwnerUpdate("alpha", next, old.NameLock, action)
				changed, e = writeOwnersBatchWith(map[string]NameLockedOwnerUpdate{root: u}, false, func(ri int, phase string, index int, stage string) error {
					if phase == "prelock" {
						compete()
					}
					return nil
				}, makeNameLockedOwnersPlan)
			} else {
				calls := 0
				changed, e = writePlanned(root, false, func(root string) (*plan, *diagnostic.Diagnostic) {
					calls++
					p, e := makeNameLockPlan(root, next, old.NameLock, action)
					if e == nil && calls == 1 {
						compete()
					}
					return p, e
				})
			}
			if e != nil || !changed {
				t.Fatal("valid intervening file+manifest update incorrectly changed state-byte CAS", named, action, changed, e)
			}
			for _, f := range next.Files {
				if string(mustRead(t, filepath.Join(root, filepath.FromSlash(f.Path)))) != f.Content {
					t.Fatal("lifecycle did not apply generated Kotlin after valid replan", f.Path)
				}
			}
		}
	}
}

func TestNameLockLifecycleGuardKeepsOrdinaryLockedWriterRepairBehavior(t *testing.T) {
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
		seedLifecycleTest(t, root, named, a)
		path := filepath.Join(root, filepath.FromSlash(a.Files[0].Path))
		if e := os.Remove(path); e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			var changed bool
			var e *diagnostic.Diagnostic
			if named {
				u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": a}, map[string][]byte{"alpha": a.NameLock}, false)
				changed, e = WriteNameLockedOwners(root, u, check)
			} else {
				changed, e = WriteNameLocked(root, a, a.NameLock, check)
			}
			if e != nil || !changed {
				t.Fatal("lifecycle-only guard changed ordinary repair", named, check, changed, e)
			}
			if check && !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("ordinary repair check wrote")
			}
		}
		if string(mustRead(t, path)) != a.Files[0].Content {
			t.Fatal("ordinary locked writer did not restore generated Kotlin")
		}
	}
}
