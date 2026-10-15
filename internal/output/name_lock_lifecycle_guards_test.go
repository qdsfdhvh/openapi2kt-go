package output

import (
	"encoding/json"
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeLifecycleTest(root string, named bool, artifact core.GenerationArtifacts, expected []byte, action NameLockAction, check bool) (bool, *diagnostic.Diagnostic) {
	if named {
		return WriteNameLockedOwners(root, lifecycleOwnerUpdate("alpha", artifact, expected, action), check)
	}
	return WriteNameLockLifecycle(root, artifact, expected, action, check)
}

func seedLifecycleTest(t testing.TB, root string, named bool, artifact core.GenerationArtifacts) string {
	t.Helper()
	if named {
		u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": artifact}, map[string][]byte{"alpha": nil}, false)
		if _, e := WriteNameLockedOwners(root, u, false); e != nil {
			t.Fatal(e)
		}
		return ownerNameLockFile("alpha")
	}
	if _, e := WriteNameLocked(root, artifact, nil, false); e != nil {
		t.Fatal(e)
	}
	return nameLockFile
}

func TestNameLockLifecycleExpectedCASAndExistingState(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	for _, named := range []bool{false, true} {
		for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
			next := lifecycleArtifacts(t, req, action)
			for _, expected := range [][]byte{nil, {}, append(append([]byte(nil), old.NameLock...), ' ')} {
				root := t.TempDir()
				seedLifecycleTest(t, root, named, old)
				before := snapshot(t, root)
				for _, check := range []bool{true, false} {
					_, e := writeLifecycleTest(root, named, next, expected, action, check)
					errorCode(t, e, "E_NAME_LOCK_STALE")
					if !reflect.DeepEqual(before, snapshot(t, root)) {
						t.Fatal("expected CAS rejection changed output", named, action)
					}
				}
			}
			for _, existingUnlocked := range []bool{false, true} {
				root := filepath.Join(t.TempDir(), "absent")
				if existingUnlocked {
					off := lifecycleArtifacts(t, req, NameLockDisable)
					if named {
						if _, e := WriteOwners(root, OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": off}}, false); e != nil {
							t.Fatal(e)
						}
					} else if _, e := Write(root, off, false); e != nil {
						t.Fatal(e)
					}
				}
				for _, check := range []bool{true, false} {
					var before map[string]snapshotEntry
					if existingUnlocked {
						before = snapshot(t, root)
					}
					_, e := writeLifecycleTest(root, named, next, nil, action, check)
					errorCode(t, e, "E_NAME_LOCK_LIFECYCLE")
					if existingUnlocked {
						if !reflect.DeepEqual(before, snapshot(t, root)) {
							t.Fatal("lifecycle silently opted in or mutated unlocked tree")
						}
					} else if _, e := os.Stat(root); !os.IsNotExist(e) {
						t.Fatal("fresh lifecycle rejection created root", e)
					}
				}
			}
		}
	}
}

func TestNameLockLifecycleArtifactAndActionGuards(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	off := lifecycleArtifacts(t, req, NameLockDisable)
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		seedLifecycleTest(t, root, named, old)
		before := snapshot(t, root)
		for _, c := range []struct {
			action NameLockAction
			a      core.GenerationArtifacts
			code   string
		}{
			{NameLockReset, off, "E_NAME_LOCK_OUTPUT_INVALID"},
			{NameLockDisable, old, "E_NAME_LOCK_OUTPUT_INVALID"},
			{NameLockReset, core.GenerationArtifacts{NameLock: old.NameLock}, "E_NAME_LOCK_OUTPUT_INVALID"},
			{NameLockDisable, core.GenerationArtifacts{}, "E_NAME_LOCK_OUTPUT_INVALID"},
			{NameLockReset, core.GenerationArtifacts{Files: old.Files, NameLock: []byte{}}, "E_NAME_LOCK_INVALID"},
			{NameLockReset, core.GenerationArtifacts{Files: old.Files, NameLock: []byte(`{}`)}, "E_NAME_LOCK_INVALID"},
			{NameLockReset, core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "../Bad.kt", Content: "synthetic"}}, NameLock: old.NameLock}, "E_OUTPUT_PATH"},
			{NameLockDisable, core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "../Bad.kt", Content: "synthetic"}}}, "E_OUTPUT_PATH"},
		} {
			for _, check := range []bool{true, false} {
				_, e := writeLifecycleTest(root, named, c.a, old.NameLock, c.action, check)
				errorCode(t, e, c.code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("invalid action/artifact mutated output", named, c.action)
				}
			}
		}
		for _, action := range []NameLockAction{"", "RESET", "unknown"} {
			code := "E_NAME_LOCK_LIFECYCLE"
			if named {
				code = "E_OWNER_UPDATE_INVALID"
			}
			for _, check := range []bool{true, false} {
				_, e := writeLifecycleTest(root, named, old, old.NameLock, action, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("unknown action mutated output")
				}
			}
		}
	}
}

func TestNameLockLifecycleResetProfileGuards(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		seedLifecycleTest(t, root, named, old)
		before := snapshot(t, root)
		for _, field := range []string{"package", "suffix", "qualified"} {
			nextReq := req
			switch field {
			case "package":
				nextReq.Options.PackageName = "org.example.other"
			case "suffix":
				nextReq.Options.ModelNameSuffix = "OtherDto"
			case "qualified":
				nextReq.Options.QualifyHoistedNames = true
			}
			next := lifecycleArtifacts(t, nextReq, NameLockReset)
			for _, check := range []bool{true, false} {
				_, e := writeLifecycleTest(root, named, next, old.NameLock, NameLockReset, check)
				errorCode(t, e, "E_NAME_LOCK_PROFILE")
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("reset migrated profile", named, field)
				}
			}
		}
	}
}

func TestNameLockLifecycleRejectsDamagedOldStateAndProtectsEvidence(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	old := singleGenerated(t, req)
	for _, named := range []bool{false, true} {
		for _, action := range []NameLockAction{NameLockReset, NameLockDisable} {
			for _, kind := range []string{"missing", "edited", "malformed", "version", "symlink", "directory", "recovery", "edited-kotlin", "handwritten-destination", "unowned-collision", "malformed-manifest"} {
				t.Run(strings.Join([]string{string(action), map[bool]string{false: "single", true: "named"}[named], kind}, "/"), func(t *testing.T) {
					root := t.TempDir()
					file := seedLifecycleTest(t, root, named, old)
					path := filepath.Join(root, file)
					code := "E_NAME_LOCK_MODIFIED"
					switch kind {
					case "missing":
						if e := os.Remove(path); e != nil {
							t.Fatal(e)
						}
						code = "E_NAME_LOCK_MISSING"
					case "edited":
						put(t, path, []byte("synthetic edited state"))
					case "malformed", "version":
						state := []byte(`{}`)
						code = "E_NAME_LOCK_INVALID"
						if kind == "version" {
							state = []byte(strings.Replace(string(old.NameLock), `"version":1`, `"version":2`, 1))
							code = "E_NAME_LOCK_VERSION"
						}
						m, e := readManifestKind(root, named, true)
						if e != nil {
							t.Fatal(e)
						}
						if named {
							(*m.NameLocks)["alpha"] = digest(state)
						} else {
							hash := digest(state)
							m.NameLock = &hash
						}
						data, e2 := json.Marshal(m)
						if e2 != nil {
							t.Fatal(e2)
						}
						put(t, filepath.Join(root, manifestName), data)
						put(t, path, state)
					case "symlink", "directory":
						if e := os.Remove(path); e != nil {
							t.Fatal(e)
						}
						if kind == "symlink" {
							put(t, filepath.Join(root, "sentinel"), old.NameLock)
							if e := os.Symlink("sentinel", path); e != nil {
								t.Fatal(e)
							}
						} else if e := os.Mkdir(path, 0700); e != nil {
							t.Fatal(e)
						}
						code = "E_OUTPUT_PATH"
					case "recovery":
						stage := filepath.Join(root, stagePrefix+"synthetic")
						if e := os.Mkdir(stage, 0700); e != nil {
							t.Fatal(e)
						}
						put(t, filepath.Join(stage, "transaction.json"), []byte("synthetic malformed journal"))
						put(t, filepath.Join(stage, "old-0"), old.NameLock)
						code = "E_RECOVERY_REQUIRED"
					case "edited-kotlin":
						put(t, filepath.Join(root, old.Files[0].Path), []byte("// synthetic handwritten edit"))
						code = "E_OUTPUT_MODIFIED"
					case "handwritten-destination":
						put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten destination"))
						code = "E_OUTPUT_COLLISION"
					case "unowned-collision":
						if e := os.Remove(filepath.Join(root, manifestName)); e != nil {
							t.Fatal(e)
						}
						code = "E_OUTPUT_COLLISION"
					case "malformed-manifest":
						put(t, filepath.Join(root, manifestName), []byte(`{}`))
						code = "E_MANIFEST_INVALID"
					}
					before := snapshot(t, root)
					next := lifecycleArtifacts(t, req, action)
					if kind == "handwritten-destination" {
						next.Files = append(next.Files, core.GeneratedFile{Path: "Handwritten.kt", Content: "// synthetic handwritten destination"})
					}
					for _, check := range []bool{true, false} {
						_, e := writeLifecycleTest(root, named, next, old.NameLock, action, check)
						errorCode(t, e, code)
						if !reflect.DeepEqual(before, snapshot(t, root)) {
							t.Fatal("lifecycle removed corrupt/missing/edited evidence")
						}
					}
				})
			}
		}
	}
}

func TestNameLockLifecycleOwnersSelectedAndExpectedInventoryGuards(t *testing.T) {
	root := t.TempDir()
	req := ownerLockRequest(t, "alpha", core.DataClass, true)
	old := ownedGenerated(t, req)
	seedLifecycleTest(t, root, true, old)
	before := snapshot(t, root)
	offReq := req
	offReq.Options.LockHoistedNames = false
	off := ownedGenerated(t, offReq)
	for _, kind := range []string{"missing-disable-expected", "missing-reset-expected", "extra-expected", "nonselected-action", "nonactive-action", "extra-empty-action", "plain-nil", "plain-nil-with-expected"} {
		u := lifecycleOwnerUpdate("alpha", off, old.NameLock, NameLockDisable)
		code := "E_OWNER_UPDATE_INVALID"
		switch kind {
		case "missing-disable-expected":
			u.PreviousNameLocks = nil
		case "missing-reset-expected":
			u.Generated["alpha"] = old
			u.NameLockActions["alpha"] = NameLockReset
			u.PreviousNameLocks = nil
		case "extra-expected":
			u.PreviousNameLocks["beta"] = nil
		case "nonselected-action":
			u.ActiveOwners = append(u.ActiveOwners, "beta")
			u.NameLockActions["beta"] = NameLockReset
		case "nonactive-action":
			u.ActiveOwners = nil
		case "extra-empty-action":
			u.NameLockActions["beta"] = ""
		case "plain-nil":
			u.NameLockActions, u.PreviousNameLocks = nil, nil
			code = "E_NAME_LOCK_DISABLED"
		case "plain-nil-with-expected":
			u.NameLockActions = nil
		}
		for _, check := range []bool{true, false} {
			_, e := WriteNameLockedOwners(root, u, check)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("invalid inventory mutated tree", kind)
			}
		}
	}
}

func TestNameLockLifecycleOwnersReconcileValidatesRemovedPeerState(t *testing.T) {
	root := t.TempDir()
	ar, br := ownerLockRequest(t, "alpha", core.DataClass, true), ownerLockRequest(t, "beta", core.DataClass, true)
	a, b := ownedGenerated(t, ar), ownedGenerated(t, br)
	initial := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
	if _, e := WriteNameLockedOwners(root, initial, false); e != nil {
		t.Fatal(e)
	}
	ar.Options.LockHoistedNames = false
	off := ownedGenerated(t, ar)
	u := lifecycleOwnerUpdate("alpha", off, a.NameLock, NameLockDisable)
	u.Reconcile = true
	put(t, filepath.Join(root, ownerNameLockFile("beta")), []byte("synthetic edited retained state"))
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		_, e := WriteNameLockedOwners(root, u, check)
		errorCode(t, e, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("disable+reconcile deleted damaged peer state")
		}
	}
	put(t, filepath.Join(root, ownerNameLockFile("beta")), b.NameLock)
	if _, e := WriteNameLockedOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	m, e := readManifestKind(root, true, true)
	if e != nil || m.Version != 4 || len(*m.NameLocks) != 0 || len(*m.Owners) != 1 || len((*m.Owners)["alpha"]) != len(a.Files) {
		t.Fatal("disable+reconcile lost retained owner or v4", m, e)
	}
	for _, owner := range []string{"alpha", "beta"} {
		if _, e := os.Stat(filepath.Join(root, ownerNameLockFile(owner))); !os.IsNotExist(e) {
			t.Fatal("disable+reconcile retained old state", owner, e)
		}
	}
	if _, e := os.Stat(filepath.Join(root, "shared", "Marker.kt")); e != nil {
		t.Fatal("disable+reconcile removed shared Kotlin", e)
	}
}
