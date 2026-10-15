package output

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func sortedNameLockTestFiles(files []core.GeneratedFile) []core.GeneratedFile {
	out := append([]core.GeneratedFile(nil), files...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func TestReadNameLockFilesDirectContentsDetachedReadonlyWithoutSidecar(t *testing.T) {
	root := t.TempDir()
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	a.Files = append(a.Files, core.GeneratedFile{Path: "synthetic/Empty.kt", Content: ""})
	seedLifecycleTest(t, root, false, a)
	if e := os.Remove(filepath.Join(root, lockName)); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten"))
	before := snapshot(t, root)
	want := sortedNameLockTestFiles(a.Files)
	files, e := ReadNameLockFiles(root, "", a.NameLock)
	if e != nil || !reflect.DeepEqual(files, want) {
		t.Fatal("direct file contents/order", files, want, e)
	}
	for i := range files {
		files[i] = core.GeneratedFile{Path: "Mutated.kt", Content: "synthetic mutation"}
	}
	files = append(files, core.GeneratedFile{Path: "Unowned.kt", Content: "synthetic mutation"})
	again, e := ReadNameLockFiles(root, "", a.NameLock)
	if e != nil || !reflect.DeepEqual(again, want) {
		t.Fatal("returned files were not detached", again, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("content read changed bytes/modes/mtimes or created a writer sidecar")
	}
}

func TestReadNameLockFilesNamedSelectedMembershipAndSharedFilesOnly(t *testing.T) {
	for _, damagePeer := range []bool{false, true} {
		root := t.TempDir()
		a := ownedGenerated(t, ownerLockRequest(t, "alpha", core.DataClass, true))
		b := ownedGenerated(t, ownerLockRequest(t, "beta", core.DataClass, true))
		u := lockedUpdate([]string{"alpha", "beta"}, map[string]core.GenerationArtifacts{"alpha": a, "beta": b}, map[string][]byte{"alpha": nil, "beta": nil}, true)
		if _, e := WriteNameLockedOwners(root, u, false); e != nil {
			t.Fatal(e)
		}
		if damagePeer {
			put(t, filepath.Join(root, ownerNameLockFile("beta")), []byte("synthetic edited peer state"))
			if e := os.Remove(filepath.Join(root, b.Files[0].Path)); e != nil {
				t.Fatal(e)
			}
		}
		if e := os.Remove(filepath.Join(root, lockName)); e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, root)
		want := sortedNameLockTestFiles(a.Files)
		files, e := ReadNameLockFiles(root, "alpha", a.NameLock)
		if e != nil || !reflect.DeepEqual(files, want) {
			t.Fatal("reader leaked peer paths/content or omitted shared membership", files, e)
		}
		shared := false
		for i := range files {
			if files[i].Path == "shared/Marker.kt" {
				shared = true
				files[i].Content = "synthetic caller mutation"
			}
		}
		if !shared {
			t.Fatal("reader omitted selected shared file")
		}
		again, e := ReadNameLockFiles(root, "alpha", a.NameLock)
		if e != nil || !reflect.DeepEqual(again, want) {
			t.Fatal("named files were not detached", again, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("selected content read mutated output")
		}
	}
}

func TestReadNameLockFilesPreservesPostMergeAliasContents(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	req.Document = []byte(`{
		"openapi":"3.1.0",
		"components":{"schemas":{"Canonical":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}}},
		"paths":{
			"/alpha":{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}}}}}}},
			"/beta":{"get":{"operationId":"beta","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}}}}}}}
		}
	}`)
	req.Options.MergeIdenticalModels = true
	a := singleGenerated(t, req)
	state, e := parseNameLockBytes(a.NameLock)
	if e != nil || len(state.Entries) != 1 {
		t.Fatal("fixture must allocate one shared locked name", state, e)
	}
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		seedLifecycleTest(t, root, named, a)
		owner := ""
		if named {
			owner = "alpha"
		}
		files, e := ReadNameLockFiles(root, owner, a.NameLock)
		if e != nil || !reflect.DeepEqual(files, sortedNameLockTestFiles(a.Files)) {
			t.Fatal("stored post-merge output differs", e)
		}
		alias := false
		for _, f := range files {
			if strings.Contains(f.Path, "/models/") && strings.Contains(f.Content, "public typealias "+state.Entries[0].Name) && strings.Contains(f.Content, "org.example.singlelock.models.CanonicalDto") {
				alias = true
				if string(mustRead(t, filepath.Join(root, f.Path))) != f.Content {
					t.Fatal("returned alias is not actual stored Kotlin")
				}
			}
		}
		if !alias {
			t.Fatal("locked pre-merge name must remain an emitted alias to the canonical component")
		}
	}
}

func TestReadNameLockFilesExpectedCASAndNoExistingState(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, named := range []bool{false, true} {
		owner := ""
		if named {
			owner = "alpha"
		}
		root := t.TempDir()
		seedLifecycleTest(t, root, named, a)
		before := snapshot(t, root)
		for _, expected := range [][]byte{nil, {}, append(append([]byte(nil), a.NameLock...), ' ')} {
			files, e := ReadNameLockFiles(root, owner, expected)
			errorCode(t, e, "E_NAME_LOCK_STALE")
			if files != nil || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("stale read returned partial files or mutated output")
			}
		}
		for _, existingUnlocked := range []bool{false, true} {
			absent := filepath.Join(t.TempDir(), "absent")
			if existingUnlocked {
				off := a
				off.NameLock = nil
				if named {
					if _, e := WriteOwners(absent, OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": off}}, false); e != nil {
						t.Fatal(e)
					}
				} else if _, e := Write(absent, off, false); e != nil {
					t.Fatal(e)
				}
			}
			var stable map[string]snapshotEntry
			if existingUnlocked {
				stable = snapshot(t, absent)
			}
			files, e := ReadNameLockFiles(absent, owner, nil)
			errorCode(t, e, "E_NAME_LOCK_LIFECYCLE")
			if files != nil {
				t.Fatal("unlocked output returned migration files")
			}
			if existingUnlocked {
				if !reflect.DeepEqual(stable, snapshot(t, absent)) {
					t.Fatal("read adopted unlocked output")
				}
			} else if _, e := os.Stat(absent); !os.IsNotExist(e) {
				t.Fatal("read created output", e)
			}
		}
	}
}

func TestReadNameLockFilesStorageGuardsAndNoPartialInventory(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, named := range []bool{false, true} {
		owner := ""
		if named {
			owner = "alpha"
		}
		for _, kind := range []string{"missing-file", "edited-file", "symlink-file", "directory-file", "missing-state", "edited-state", "symlink-state", "directory-state", "malformed-owned-state", "recovery", "unsafe-manifest-path", "malformed-manifest", "unowned-state"} {
			t.Run(map[bool]string{false: "direct/", true: "named/"}[named]+kind, func(t *testing.T) {
				root := t.TempDir()
				stateFile := seedLifecycleTest(t, root, named, a)
				// A late path failure must not return the files already inspected.
				file := a.Files[len(a.Files)-1]
				path := filepath.Join(root, filepath.FromSlash(file.Path))
				code := "E_OUTPUT_PATH"
				switch kind {
				case "missing-file":
					if e := os.Remove(path); e != nil {
						t.Fatal(e)
					}
					code = "E_OUTPUT_MISSING"
				case "edited-file":
					put(t, path, []byte("// synthetic edited Kotlin"))
					code = "E_OUTPUT_MODIFIED"
				case "symlink-file", "directory-file":
					if e := os.Remove(path); e != nil {
						t.Fatal(e)
					}
					if kind == "symlink-file" {
						sentinel := filepath.Join(root, "sentinel")
						put(t, sentinel, []byte(file.Content))
						if e := os.Symlink(sentinel, path); e != nil {
							t.Fatal(e)
						}
					} else if e := os.Mkdir(path, 0700); e != nil {
						t.Fatal(e)
					}
				case "missing-state":
					if e := os.Remove(filepath.Join(root, stateFile)); e != nil {
						t.Fatal(e)
					}
					code = "E_NAME_LOCK_MISSING"
				case "edited-state":
					put(t, filepath.Join(root, stateFile), []byte("synthetic edited state"))
					code = "E_NAME_LOCK_MODIFIED"
				case "symlink-state", "directory-state":
					statePath := filepath.Join(root, stateFile)
					if e := os.Remove(statePath); e != nil {
						t.Fatal(e)
					}
					if kind == "symlink-state" {
						sentinel := filepath.Join(root, "state-sentinel")
						put(t, sentinel, a.NameLock)
						if e := os.Symlink(sentinel, statePath); e != nil {
							t.Fatal(e)
						}
					} else if e := os.Mkdir(statePath, 0700); e != nil {
						t.Fatal(e)
					}
				case "malformed-owned-state":
					state := []byte(`{}`)
					m, e := readManifestKind(root, named, true)
					if e != nil {
						t.Fatal(e)
					}
					if named {
						(*m.NameLocks)[owner] = digest(state)
					} else {
						hash := digest(state)
						m.NameLock = &hash
					}
					data, e2 := json.Marshal(m)
					if e2 != nil {
						t.Fatal(e2)
					}
					put(t, filepath.Join(root, manifestName), data)
					put(t, filepath.Join(root, stateFile), state)
					code = "E_NAME_LOCK_INVALID"
				case "recovery":
					stage := filepath.Join(root, stagePrefix+"synthetic")
					if e := os.Mkdir(stage, 0700); e != nil {
						t.Fatal(e)
					}
					put(t, filepath.Join(stage, "transaction.json"), []byte("synthetic malformed journal"))
					put(t, filepath.Join(stage, "old-0"), a.NameLock)
					code = "E_RECOVERY_REQUIRED"
				case "unsafe-manifest-path":
					m, e := readManifestKind(root, named, true)
					if e != nil {
						t.Fatal(e)
					}
					m.Files["../Unsafe.kt"] = digest([]byte("synthetic"))
					data, e2 := json.Marshal(m)
					if e2 != nil {
						t.Fatal(e2)
					}
					put(t, filepath.Join(root, manifestName), data)
				case "malformed-manifest":
					put(t, filepath.Join(root, manifestName), []byte(`{}`))
					code = "E_MANIFEST_INVALID"
				case "unowned-state":
					if e := os.Remove(filepath.Join(root, manifestName)); e != nil {
						t.Fatal(e)
					}
					code = "E_OUTPUT_COLLISION"
				}
				before := snapshot(t, root)
				files, e := ReadNameLockFiles(root, owner, a.NameLock)
				errorCode(t, e, code)
				if files != nil || !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("unsafe read returned partial files or changed evidence")
				}
			})
		}
	}
}

func TestReadNameLockFilesRootOwnerAndParentGuards(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	for _, c := range []struct{ root, owner, code string }{
		{root, "../alpha", "E_OWNER_UPDATE_INVALID"},
		{"", "", "E_OUTPUT_PATH"},
		{root + string(filepath.Separator) + ".." + string(filepath.Separator) + "unsafe", "alpha", "E_OUTPUT_PATH"},
	} {
		files, e := ReadNameLockFiles(c.root, c.owner, nil)
		errorCode(t, e, c.code)
		if files != nil {
			t.Fatal("unsafe root/owner returned files")
		}
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("unsafe root/owner read created output", e)
	}
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
		seedLifecycleTest(t, root, named, a)
		owner := ""
		if named {
			owner = "alpha"
		}
		beforeRoot := snapshot(t, root)
		alias := filepath.Join(t.TempDir(), "root-alias")
		if e := os.Symlink(root, alias); e != nil {
			t.Fatal(e)
		}
		files, e := ReadNameLockFiles(alias, owner, a.NameLock)
		errorCode(t, e, "E_OUTPUT_PATH")
		if files != nil || !reflect.DeepEqual(beforeRoot, snapshot(t, root)) {
			t.Fatal("root symlink rejection changed target")
		}
		parent := filepath.Dir(filepath.Join(root, a.Files[0].Path))
		moved := filepath.Join(t.TempDir(), "moved")
		if e := os.Rename(parent, moved); e != nil {
			t.Fatal(e)
		}
		if e := os.Symlink(moved, parent); e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, moved)
		files, e = ReadNameLockFiles(root, owner, a.NameLock)
		errorCode(t, e, "E_OUTPUT_PATH")
		link, err := os.Readlink(parent)
		if files != nil || err != nil || link != moved || !reflect.DeepEqual(before, snapshot(t, moved)) {
			t.Fatal("parent symlink rejection changed target", err)
		}
	}
}
