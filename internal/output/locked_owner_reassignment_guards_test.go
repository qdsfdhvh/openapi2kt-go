package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLockedOwnerReassignmentWholeOmittedPeerCAS(t *testing.T) {
	for _, kind := range []string{"manifest", "code", "raw-lock", "valid-raw-lock", "mode", "mtime", "foreign", "recovery"} {
		t.Run(kind, func(t *testing.T) {
			root, u := reassignmentFixture(t, []string{"newowner"})
			before := snapshot(t, root)
			code := "E_LOCKED_OWNER_REASSIGN_STALE"
			_, e := writeOwnersBatchWith(map[string]LockedOwnerReassignmentUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
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
					put(t, filepath.Join(root, ownerNameLockFile("left")), []byte("intervening lock"))
					code = "E_NAME_LOCK_MODIFIED"
				case "valid-raw-lock":
					raw := append(u.Previous.Previous.NameLocks["left"], ' ')
					put(t, filepath.Join(root, ownerNameLockFile("left")), raw)
					m, e := readManifestKind(root, true, true)
					if e != nil {
						t.Fatal(e)
					}
					(*m.NameLocks)["left"] = digest(raw)
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
			}, makeLockedOwnerReassignmentPlan)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("omitted peer CAS destroyed intervening state", kind)
			}
		})
	}
}

func TestLockedOwnerReassignmentOriginalUnionRejectsProfilesAndNames(t *testing.T) {
	for _, kind := range []string{"profile", "allocation-name", "role-origin", "foreign", "missing-code", "unlocked-owner"} {
		root, _ := reassignmentFixture(t, []string{"left", "right"})
		m, e := readManifestKind(root, true, true)
		if e != nil {
			t.Fatal(e)
		}
		switch kind {
		case "foreign":
			put(t, filepath.Join(root, ownerNameLockFile("foreign")), []byte("synthetic foreign"))
		case "missing-code":
			for path := range m.Files {
				if e := os.Remove(filepath.Join(root, path)); e != nil {
					t.Fatal(e)
				}
				break
			}
		case "unlocked-owner":
			delete(*m.NameLocks, "right")
			if e := os.Remove(filepath.Join(root, ownerNameLockFile("right"))); e != nil {
				t.Fatal(e)
			}
			data, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(root, manifestName), data)
		default:
			raw := mustRead(t, filepath.Join(root, ownerNameLockFile("right")))
			state, e := parseNameLockBytes(raw)
			if e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "profile":
				state.Profile.Package = "org.example.other"
			case "allocation-name":
				state.Entries[0].Name += "Other"
			case "role-origin":
				left, e := parseNameLockBytes(mustRead(t, filepath.Join(root, ownerNameLockFile("left"))))
				if e != nil {
					t.Fatal(e)
				}
				state.Entries[0].Shape = string(bytes.Repeat([]byte("a"), 64))
				state.Entries[0].Name = "OtherOccurrence"
				state.Entries[0].Occurrences = left.Entries[0].Occurrences
			}
			raw, e = state.Encode()
			if e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(root, ownerNameLockFile("right")), raw)
			(*m.NameLocks)["right"] = digest(raw)
			data, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(root, manifestName), data)
		}
		before := snapshot(t, root)
		_, e = ReadLockedOwnerReassignmentState(root, []string{"left", "right"})
		if e == nil {
			t.Fatal("invalid original union accepted", kind)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("union observation wrote", kind)
		}
	}
}
