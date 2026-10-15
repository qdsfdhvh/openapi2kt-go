package output

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"openapi-kt/core"
)

func lockedAdoptionFixture(t testing.TB) (string, LockedOwnerAdoptionUpdate) {
	t.Helper()
	root := t.TempDir()
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	if _, e := WriteNameLocked(root, a, nil, false); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, "Handwritten.kt"), []byte("// synthetic handwritten control"))
	put(t, filepath.Join(root, ownerNameLockFile("foreign")), []byte("synthetic foreign metadata"))
	for _, path := range []string{manifestName, nameLockFile, a.Files[0].Path} {
		file := filepath.Join(root, path)
		if e := os.Chmod(file, 0640); e != nil {
			t.Fatal(e)
		}
		stamp := time.Unix(1234567890, 123456000)
		if e := os.Chtimes(file, stamp, stamp); e != nil {
			t.Fatal(e)
		}
	}
	s, e := ReadLockedOwnerAdoptionState(root, "catalog")
	if e != nil {
		t.Fatal(e)
	}
	return root, LockedOwnerAdoptionUpdate{Owner: "catalog", Previous: s, Generated: a}
}

func TestLockedOwnerAdoptionRawMetadataOnlyNoop(t *testing.T) {
	root, u := lockedAdoptionFixture(t)
	// Raw bytes need not be the core's canonical serialization. Preserve them.
	raw := append(append([]byte(" \n"), u.Previous.NameLock...), '\n')
	put(t, filepath.Join(root, nameLockFile), raw)
	m, e := readManifestKind(root, false, true)
	if e != nil {
		t.Fatal(e)
	}
	hash := digest(raw)
	m.NameLock = &hash
	data, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), data)
	u.Previous, e = ReadLockedOwnerAdoptionState(root, u.Owner)
	if e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	rootInfo, _ := os.Stat(root)
	p, e := makeLockedOwnerAdoptionPlan(u.Previous.Root, u)
	if e != nil || !p.Changed || len(p.Writes) != 1 || !reflect.DeepEqual(p.Removals, []string{nameLockFile}) {
		t.Fatal("non metadata plan", p, e)
	}
	proposed, e := LockedOwnerAdoptionManifest(u.Previous, u.Owner)
	if e != nil {
		t.Fatal(e)
	}
	if changed, e := WriteLockedOwnerAdoption(root, u, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	now, _ := os.Stat(root)
	if !reflect.DeepEqual(before, snapshot(t, root)) || !rootInfo.ModTime().Equal(now.ModTime()) {
		t.Fatal("check wrote tree or directory")
	}
	if changed, e := WriteLockedOwnerAdoption(root, u, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	after := snapshot(t, root)
	if !bytes.Equal(proposed, mustRead(t, filepath.Join(root, manifestName))) || !bytes.Equal(raw, mustRead(t, filepath.Join(root, ownerNameLockFile(u.Owner)))) {
		t.Fatal("report/raw byte binding")
	}
	if _, e := os.Stat(filepath.Join(root, nameLockFile)); !os.IsNotExist(e) {
		t.Fatal("orphan anonymous payload", e)
	}
	for _, path := range sortedKeys(before) {
		entry := before[path]
		if path != manifestName && path != nameLockFile && !reflect.DeepEqual(entry, after[path]) {
			t.Fatal("Kotlin/unowned attrs changed", path)
		}
		if filepath.Ext(path) == ".kt" || path == ownerNameLockFile("foreign") {
			next := after[path]
			t.Logf("preserved path=%s sha256=%s -> %s mode=%#o -> %#o mtime=%s -> %s", path, digest([]byte(entry.Data)), digest([]byte(next.Data)), entry.Mode.Perm(), next.Mode.Perm(), entry.Modified.UTC().Format(time.RFC3339Nano), next.Modified.UTC().Format(time.RFC3339Nano))
		}
	}
	t.Logf("metadata-only manifest v3 -> v4 sha256=%s -> %s; raw lock sha256=%s -> %s", digest(u.Previous.Manifest), digest([]byte(after[manifestName].Data)), digest(raw), digest([]byte(after[ownerNameLockFile(u.Owner)].Data)))
	m, e = readManifestKind(root, true, true)
	if e != nil || m.Version != 4 || len(*m.Owners) != 1 || len(*m.NameLocks) != 1 || len((*m.Owners)[u.Owner]) != len(m.Files) || (*m.NameLocks)[u.Owner] != hash {
		t.Fatal("complete ownership", m, e)
	}
	u.Previous, e = ReadLockedOwnerAdoptionState(root, u.Owner)
	if e != nil {
		t.Fatal(e)
	}
	for _, check := range []bool{true, false} {
		if changed, e := WriteLockedOwnerAdoption(root, u, check); e != nil || changed {
			t.Fatal("no-op replay", changed, e)
		}
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("no-op rewrote raw payload")
		}
	}
	_, e = ReadLockedOwnerAdoptionState(root, "other")
	errorCode(t, e, "E_LOCKED_OWNER_ADOPTION_INVALID")
}

func TestLockedOwnerAdoptionCandidateAndSourceGuards(t *testing.T) {
	for _, kind := range []string{"membership", "duplicate", "sourcebytes", "disabled", "allocations", "occurrences", "profile", "missing", "edited", "lockmissing", "lockedited", "invalidlock", "lockversion", "manifest", "v1", "v2", "occupied", "recovery", "manifestcas", "lockcas", "filecas", "otherroot", "owner"} {
		t.Run(kind, func(t *testing.T) {
			root, u := lockedAdoptionFixture(t)
			code := "E_LOCKED_OWNER_ADOPTION_CHANGED"
			switch kind {
			case "membership":
				u.Generated.Files = u.Generated.Files[1:]
			case "duplicate":
				u.Generated.Files = append(u.Generated.Files, u.Generated.Files[0])
			case "sourcebytes":
				u.Generated.Files[0].Content += "\n// drift"
			case "disabled":
				u.Generated.NameLock = nil
			case "allocations":
				u.Generated.NameLock = singleGenerated(t, addedSinglePeer(t, singleLockRequest(t, core.DataClass, true))).NameLock
			case "occurrences":
				state, e := parseNameLockBytes(u.Generated.NameLock)
				if e != nil {
					t.Fatal(e)
				}
				state.Entries[0].Occurrences[0] += "/extra"
				u.Generated.NameLock, _ = json.Marshal(state)
			case "profile":
				state, e := parseNameLockBytes(u.Generated.NameLock)
				if e != nil {
					t.Fatal(e)
				}
				state.Profile.Qualified = !state.Profile.Qualified
				u.Generated.NameLock, _ = json.Marshal(state)
				code = "E_NAME_LOCK_PROFILE"
			case "missing":
				if e := os.Remove(filepath.Join(root, u.Generated.Files[0].Path)); e != nil {
					t.Fatal(e)
				}
				code = "E_OUTPUT_MISSING"
			case "edited":
				put(t, filepath.Join(root, u.Generated.Files[0].Path), []byte("edited"))
				code = "E_OUTPUT_MODIFIED"
			case "lockmissing":
				if e := os.Remove(filepath.Join(root, nameLockFile)); e != nil {
					t.Fatal(e)
				}
				code = "E_NAME_LOCK_MISSING"
			case "lockedited":
				put(t, filepath.Join(root, nameLockFile), []byte("edited"))
				code = "E_NAME_LOCK_MODIFIED"
			case "invalidlock", "lockversion":
				raw := []byte(`{"version":1,"profile":{},"entries":[]}`)
				code = "E_NAME_LOCK_INVALID"
				if kind == "lockversion" {
					raw = bytes.Replace(u.Generated.NameLock, []byte(`"version":1`), []byte(`"version":99`), 1)
					code = "E_NAME_LOCK_VERSION"
				}
				put(t, filepath.Join(root, nameLockFile), raw)
				m, e := readManifestKind(root, false, true)
				if e != nil {
					t.Fatal(e)
				}
				hash := digest(raw)
				m.NameLock = &hash
				data, _ := json.Marshal(m)
				put(t, filepath.Join(root, manifestName), data)
			case "manifest":
				put(t, filepath.Join(root, manifestName), []byte(`{"version":3,"version":3}`))
				code = "E_MANIFEST_INVALID"
			case "v1", "v2":
				m, e := readManifestKind(root, false, true)
				if e != nil {
					t.Fatal(e)
				}
				m.Version = 1
				m.NameLock = nil
				if kind == "v2" {
					m.Version = 2
					owners := map[string][]string{"catalog": sortedKeys(m.Files)}
					m.Owners = &owners
				}
				data, _ := json.Marshal(m)
				put(t, filepath.Join(root, manifestName), data)
				code = "E_LOCKED_OWNER_ADOPTION_INVALID"
			case "occupied":
				put(t, filepath.Join(root, ownerNameLockFile(u.Owner)), u.Generated.NameLock)
				code = "E_OUTPUT_COLLISION"
			case "recovery":
				if e := os.Mkdir(filepath.Join(root, stagePrefix+"retained"), 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(root, stagePrefix+"retained", "transaction.json"), []byte("malformed"))
				code = "E_RECOVERY_REQUIRED"
			case "manifestcas":
				u.Previous.Manifest = append(u.Previous.Manifest, ' ')
				code = "E_LOCKED_OWNER_ADOPTION_STALE"
			case "lockcas":
				u.Previous.NameLock = append(u.Previous.NameLock, ' ')
				code = "E_LOCKED_OWNER_ADOPTION_STALE"
			case "filecas":
				u.Previous.Files = append([]core.GeneratedFile(nil), u.Previous.Files...)
				u.Previous.Files[0].Content += " "
				code = "E_LOCKED_OWNER_ADOPTION_STALE"
			case "owner":
				u.Owner = "other"
				code = "E_LOCKED_OWNER_ADOPTION_INVALID"
			case "otherroot":
				root, _ = lockedAdoptionFixture(t)
				code = "E_LOCKED_OWNER_ADOPTION_STALE"
			}
			before := snapshot(t, root)
			for _, check := range []bool{true, false} {
				_, e := WriteLockedOwnerAdoption(root, u, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("guard lost old output", kind)
				}
			}
		})
	}
	absent := filepath.Join(t.TempDir(), "absent")
	_, e := ReadLockedOwnerAdoptionState(absent, "catalog")
	errorCode(t, e, "E_LOCKED_OWNER_ADOPTION_INVALID")
	if _, e := os.Stat(absent); !os.IsNotExist(e) {
		t.Fatal("read created missing root")
	}
}

func TestLockedOwnerAdoptionEveryBoundaryAndRecovery(t *testing.T) {
	updates := map[string]LockedOwnerAdoptionUpdate{}
	before := map[string]map[string]snapshotEntry{}
	for i := 0; i < 2; i++ {
		root, u := lockedAdoptionFixture(t)
		updates[root] = u
		before[root] = snapshot(t, root)
	}
	roots := sortedKeys(updates)
	restored := func() {
		t.Helper()
		for root, old := range before {
			if !reflect.DeepEqual(old, snapshot(t, root)) {
				t.Fatal("failed rollback bytes/mode/mtime", root)
			}
		}
	}
	for ri := range roots {
		for boundary := -1; boundary < 3; boundary++ {
			_, e := writeOwnersBatchWith(updates, false, func(r int, phase string, op int, stage string) error {
				if phase == "prepare" && boundary == -1 && r == ri {
					return errors.New("synthetic preparation fault")
				}
				if phase == "apply" {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if len(j.Operations) != 3 || j.Operations[2].Path != manifestName {
						t.Fatal("manifest not last")
					}
					for _, entry := range j.Operations {
						if filepath.Ext(entry.Path) == ".kt" {
							t.Fatal("Kotlin journal operation")
						}
					}
					for _, root := range roots {
						stages, _ := filepath.Glob(filepath.Join(root, stagePrefix+"*"))
						if len(stages) != 1 {
							t.Fatal("all roots not prepared")
						}
					}
					if r == ri && op == boundary {
						return errors.New("synthetic apply fault")
					}
				}
				return nil
			}, makeLockedOwnerAdoptionPlan)
			code := "E_OUTPUT_COMMIT"
			if boundary == -1 {
				code = "E_OUTPUT_IO"
			}
			errorCode(t, e, code)
			restored()
		}
	}
	_, e := writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, stage string) error {
		if phase == "apply" && ri == 1 && op == 2 {
			obstructLifecycleRollback(t, roots[0], nameLockFile)
			return errors.New("synthetic rollback obstruction")
		}
		return nil
	}, makeLockedOwnerAdoptionPlan)
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, roots[0], nameLockFile, updates[roots[0]].Previous.NameLock)
	// Rollback still restores manifest/new destination and the other root.
	if !bytes.Equal(mustRead(t, filepath.Join(roots[0], manifestName)), updates[roots[0]].Previous.Manifest) {
		t.Fatal("rest of blocked root rollback not attempted")
	}
	if !reflect.DeepEqual(before[roots[1]], snapshot(t, roots[1])) {
		t.Fatal("other root rollback skipped")
	}
	retained := snapshot(t, roots[0])
	for _, check := range []bool{true, false} {
		_, e := WriteLockedOwnerAdoption(roots[0], updates[roots[0]], check)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(retained, snapshot(t, roots[0])) {
			t.Fatal("retained evidence lost")
		}
	}
	_, e = ReadLockedOwnerAdoptionState(roots[0], "catalog")
	errorCode(t, e, "E_RECOVERY_REQUIRED")
}

func TestLockedOwnerAdoptionAggregateKotlinBudget(t *testing.T) {
	for _, test := range []struct {
		name       string
		extra      int64
		beforeHash bool
	}{{"boundary", 0, false}, {"overflow", 1, false}, {"overflow-before-hash", 1, true}} {
		t.Run(test.name, func(t *testing.T) {
			root, u := lockedAdoptionFixture(t)
			m, e := readManifestKind(root, false, true)
			if e != nil {
				t.Fatal(e)
			}
			m.Files = map[string]string{}
			// Sparse synthetic files exercise the real 64 MiB boundary with bounded
			// streaming fixture construction rather than a whole-file buffer.
			for i, name := range []string{"ResourceA.kt", "ResourceB.kt"} {
				file, err := os.Create(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				size := int64(maxAdoptionKotlinBytes / 2)
				if i == 1 {
					size += test.extra
				}
				if err := file.Truncate(size); err != nil {
					t.Fatal(err)
				}
				hash := sha256.New()
				if _, err := io.Copy(hash, file); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				m.Files[name] = hex.EncodeToString(hash.Sum(nil))
			}
			if test.beforeHash {
				// Resource limits must reject the complete inventory before content
				// reads/hashing, even if its first payload would fail its hash check.
				m.Files["ResourceA.kt"] = strings.Repeat("0", 64)
			}
			data, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(root, manifestName), data)
			before, err := os.Stat(filepath.Join(root, manifestName))
			if err != nil {
				t.Fatal(err)
			}
			state, e := ReadLockedOwnerAdoptionState(root, u.Owner)
			if test.extra == 1 {
				errorCode(t, e, "E_OUTPUT_LIMIT")
				t.Logf("old Kotlin total=%d limit=%d diagnostic=%s; before_payload_hash=%t", int64(maxAdoptionKotlinBytes)+test.extra, maxAdoptionKotlinBytes, e.Code, test.beforeHash)
			} else if e != nil || len(state.Files) != 2 {
				t.Fatal("exact aggregate boundary", e)
			} else {
				t.Logf("old Kotlin total=%d limit=%d accepted files=%d", maxAdoptionKotlinBytes, maxAdoptionKotlinBytes, len(state.Files))
			}
			after, err := os.Stat(filepath.Join(root, manifestName))
			if err != nil || !before.ModTime().Equal(after.ModTime()) || !bytes.Equal(data, mustRead(t, filepath.Join(root, manifestName))) {
				t.Fatal("budget read changed metadata", err)
			}
		})
	}
}

func TestLockedOwnerAdoptionMissingStagedManifestRestoresBackup(t *testing.T) {
	root, u := lockedAdoptionFixture(t)
	before := snapshot(t, root)
	_, e := writeOwnersBatchWith(map[string]LockedOwnerAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, stage string) error {
		if phase == "apply" && op == 2 {
			if e := os.Remove(filepath.Join(stage, "new-2")); e != nil {
				t.Fatal(e)
			}
		}
		return nil
	}, makeLockedOwnerAdoptionPlan)
	errorCode(t, e, "E_OUTPUT_COMMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("failed install after manifest backup lost attrs")
	}
}
