package output

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRetainedLockedAdoptionWholeOriginalCAS(t *testing.T) {
	for _, kind := range []string{"manifest", "code", "raw-lock", "valid-raw-lock", "mode", "mtime", "foreign", "recovery"} {
		t.Run(kind, func(t *testing.T) {
			root, u := retainedAdoptionFixture(t, []string{"part"})
			before := snapshot(t, root)
			code := "E_RETAINED_LOCKED_ADOPTION_STALE"
			_, e := writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
				if phase != "prelock" {
					return nil
				}
				path := u.Previous.Previous.Files[0].Path
				switch kind {
				case "manifest":
					put(t, filepath.Join(root, manifestName), append(u.Previous.Previous.Manifest, ' '))
				case "code":
					put(t, filepath.Join(root, path), []byte("intervening Kotlin"))
					code = "E_OUTPUT_MODIFIED"
				case "raw-lock":
					put(t, filepath.Join(root, nameLockFile), []byte("intervening lock"))
					code = "E_NAME_LOCK_MODIFIED"
				case "valid-raw-lock":
					raw := append(u.Previous.NameLock, ' ')
					put(t, filepath.Join(root, nameLockFile), raw)
					m, e := readManifestKind(root, false, true)
					if e != nil {
						t.Fatal(e)
					}
					hash := digest(raw)
					m.NameLock = &hash
					data, _ := json.MarshalIndent(m, "", "  ")
					put(t, filepath.Join(root, manifestName), data)
				case "mode":
					old := u.Previous.Previous.Entries[path].Attributes.Mode
					mode := os.FileMode(0600)
					if old == mode {
						mode = 0640
					}
					if e := os.Chmod(filepath.Join(root, path), mode); e != nil {
						t.Fatal(e)
					}
					info, err := os.Stat(filepath.Join(root, path))
					if err != nil {
						t.Fatal(err)
					}
					if info.Mode().Perm() == old {
						t.Skip("filesystem does not expose distinct permission bits")
					}

				case "mtime":
					stamp := time.Unix(1400000000, 0)
					if e := os.Chtimes(filepath.Join(root, path), stamp, stamp); e != nil {
						t.Fatal(e)
					}
				case "foreign":
					put(t, filepath.Join(root, ownerNameLockFile("foreign")), []byte("synthetic foreign"))
					code = "E_OUTPUT_ROOT_MOVE_INVALID"
				case "recovery":
					if e := os.Mkdir(filepath.Join(root, stagePrefix+"existing"), 0700); e != nil {
						t.Fatal(e)
					}
					code = "E_RECOVERY_REQUIRED"
				}
				before = snapshot(t, root)
				return nil
			}, makeRetainedLockedAdoptionPlan)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("omitted peer CAS destroyed intervening state", kind)
			}
		})
	}
}

func TestRetainedLockedAdoptionUnselectedV4PeerCAS(t *testing.T) {
	root, u := retainedAdoptionFixture(t, []string{"part"})
	if _, e := WriteRetainedLockedAdoption(root, u, false); e != nil {
		t.Fatal(e)
	}
	s, e := ReadRetainedLockedAdoptionState(root, "original", []string{"extra"})
	if e != nil {
		t.Fatal(e)
	}
	u = RetainedLockedAdoptionUpdate{Previous: s, Generated: map[string]core.GenerationArtifacts{"extra": u.Generated["part"]}}
	before := snapshot(t, root)
	_, e = writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			raw := append(s.Previous.NameLocks["part"], ' ')
			put(t, filepath.Join(root, ownerNameLockFile("part")), raw)
			m, e := readManifestKind(root, true, true)
			if e != nil {
				t.Fatal(e)
			}
			(*m.NameLocks)["part"] = digest(raw)
			data, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(root, manifestName), data)
			before = snapshot(t, root)
		}
		return nil
	}, makeRetainedLockedAdoptionPlan)
	errorCode(t, e, "E_RETAINED_LOCKED_ADOPTION_STALE")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("unselected existing peer CAS destroyed concurrent state")
	}
}
