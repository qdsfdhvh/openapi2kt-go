package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/schema"
)

func existingLockedFixture(t testing.TB) (string, ExistingLockedAdoptionUpdate) {
	t.Helper()
	root, old := splitLockedFixture(t)
	if _, err := WriteSplitLockedAdoption(root, old, false); err != nil {
		t.Fatal(err)
	}
	prior, err := ReadExistingLockedAdoptionState(root, "consumer", "left")
	if err != nil {
		t.Fatal(err)
	}
	return root, ExistingLockedAdoptionUpdate{Previous: prior, Generated: old.Generated["left"]}
}

func TestExistingLockedAdoptionLimitsIncludePeersAndNewOwner(t *testing.T) {
	t.Run("owners", func(t *testing.T) {
		root, u := existingLockedFixture(t)
		m, e := readManifestKind(root, true, true)
		if e != nil {
			t.Fatal(e)
		}
		payload := u.Previous.Previous.NameLocks["left"]
		for i := 2; i < maxLockedOwners; i++ {
			name := fmt.Sprintf("peer%03d", i)
			(*m.Owners)[name] = append([]string{}, (*m.Owners)["left"]...)
			(*m.NameLocks)[name] = digest(payload)
			put(t, filepath.Join(root, ownerNameLockFile(name)), payload)
		}
		raw, _ := json.Marshal(m)
		put(t, filepath.Join(root, manifestName), raw)
		before := snapshot(t, root)
		_, e = ReadExistingLockedAdoptionState(root, "consumer", "left")
		errorCode(t, e, "E_LOCKED_OWNER_ADOPTION_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("257th owner changed root")
		}
	})
	t.Run("locks", func(t *testing.T) {
		root, u := existingLockedFixture(t)
		m, e := readManifestKind(root, true, true)
		if e != nil {
			t.Fatal(e)
		}
		payload := u.Previous.Previous.NameLocks["left"]
		payload = append(append([]byte{}, payload...), bytes.Repeat([]byte(" "), schema.MaxNameLockBytes-len(payload))...)
		for i := 0; i < 16; i++ {
			name := fmt.Sprintf("peer%02d", i)
			if i == 0 {
				name = "left"
			}
			if i == 1 {
				name = "right"
			}
			(*m.Owners)[name] = append([]string{}, (*m.Owners)["left"]...)
			(*m.NameLocks)[name] = digest(payload)
			put(t, filepath.Join(root, ownerNameLockFile(name)), payload)
		}
		raw, _ := json.Marshal(m)
		put(t, filepath.Join(root, manifestName), raw)
		u.Previous, e = ReadExistingLockedAdoptionState(root, "consumer", "left")
		if e != nil {
			t.Fatal("16MiB read boundary", e)
		}
		u.Generated.NameLock = payload
		before := snapshot(t, root)
		_, e = WriteExistingLockedAdoption(root, u, false)
		errorCode(t, e, "E_NAME_LOCK_LIMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("new lock exceeded peer-inclusive limit")
		}
	})
	t.Run("source-size-first", func(t *testing.T) {
		root, u := existingLockedFixture(t)
		if err := os.Truncate(filepath.Join(root, u.Previous.Previous.Files[0].Path), int64(maxAdoptionKotlinBytes)+1); err != nil {
			t.Fatal(err)
		}
		_, e := ReadExistingLockedAdoptionState(root, "consumer", "left")
		errorCode(t, e, "E_OUTPUT_LIMIT")
	})
}

func TestExistingLockedAdoptionMissingManifestStageAndRecovery(t *testing.T) {
	for _, obstruct := range []bool{false, true} {
		root, u := existingLockedFixture(t)
		before := snapshot(t, root)
		_, e := writeOwnersBatchWith(map[string]ExistingLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, stage string) error {
			if phase == "apply" && op == 1 {
				if err := os.Remove(filepath.Join(stage, "new-1")); err != nil {
					t.Fatal(err)
				}
				if obstruct {
					obstructLifecycleRollback(t, root, ownerNameLockFile("consumer"))
				}
			}
			return nil
		}, makeExistingLockedAdoptionPlan)
		if !obstruct {
			errorCode(t, e, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("manifest backup lost after staged rename failure")
			}
			continue
		}
		errorCode(t, e, "E_RECOVERY_REQUIRED")
		if !bytes.Equal(mustRead(t, filepath.Join(root, manifestName)), u.Previous.Previous.Manifest) {
			t.Fatal("unobstructed manifest rollback skipped")
		}
		stages, err := filepath.Glob(filepath.Join(root, stagePrefix+"*"))
		if err != nil || len(stages) != 1 {
			t.Fatal("missing recovery evidence", stages, err)
		}
		_ = mustRead(t, filepath.Join(stages[0], "transaction.json"))
		retained := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteExistingLockedAdoption(root, u, check)
			errorCode(t, e, "E_RECOVERY_REQUIRED")
			if !reflect.DeepEqual(retained, snapshot(t, root)) {
				t.Fatal("retained evidence changed")
			}
		}
	}
}

func TestExistingLockedAdoptionSubsetRawPeersAndReplay(t *testing.T) {
	root, u := existingLockedFixture(t)
	// Noncanonical peer/donor bytes remain authoritative even when the core
	// would emit canonical JSON. Include a distinct peer lock profile.
	m, err := readManifestKind(root, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"left", "right"} {
		payload := mustRead(t, filepath.Join(root, ownerNameLockFile(name)))
		if name == "right" {
			state, err := parseNameLockBytes(payload)
			if err != nil {
				t.Fatal(err)
			}
			state.Profile.Package = "org.example.peer"
			payload, err = state.Encode()
			if err != nil {
				t.Fatal(err)
			}
		}
		payload = append(append([]byte(" \n"), payload...), '\n')
		put(t, filepath.Join(root, ownerNameLockFile(name)), payload)
		(*m.NameLocks)[name] = digest(payload)
	}
	raw, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), raw)
	u.Previous, err = ReadExistingLockedAdoptionState(root, "consumer", "left")
	if err != nil {
		t.Fatal(err)
	}
	u.Generated.Files = append([]core.GeneratedFile(nil), u.Generated.Files[1:]...)
	if len(u.Generated.Files) == 0 {
		t.Fatal("fixture lacks subset")
	}
	before := snapshot(t, root)
	if changed, err := WriteExistingLockedAdoption(root, u, true); err != nil || !changed || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal(changed, err)
	}
	p, err := makeExistingLockedAdoptionPlan(u.Previous.Previous.Root, u)
	if err != nil || !p.Changed || len(p.Writes) != 1 || len(p.Removals) != 0 {
		t.Fatal(p, err)
	}
	if changed, err := WriteExistingLockedAdoption(root, u, false); err != nil || !changed {
		t.Fatal(changed, err)
	}
	after := snapshot(t, root)
	for path, old := range before {
		if path != manifestName && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("old inventory modified", path)
		}
	}
	if !bytes.Equal(mustRead(t, filepath.Join(root, ownerNameLockFile("consumer"))), u.Previous.Previous.NameLocks["left"]) {
		t.Fatal("lost donor raw bytes")
	}
	next, err := readManifestKind(root, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for owner, paths := range u.Previous.Previous.Owners {
		if !reflect.DeepEqual((*next.Owners)[owner], paths) {
			t.Fatal("redistributed donor/peer memberships")
		}
	}
	if len((*next.Owners)["consumer"]) != len(u.Generated.Files) {
		t.Fatal("incorrect subset membership")
	}
	u.Previous, err = ReadExistingLockedAdoptionState(root, "consumer", "left")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []bool{true, false} {
		if changed, err := WriteExistingLockedAdoption(root, u, check); err != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("replay wrote metadata", changed, err)
		}
	}
	u.Generated.Files = u.Generated.Files[1:]
	if _, err := WriteExistingLockedAdoption(root, u, false); err == nil {
		t.Fatal("replay changed membership")
	}
}

func TestExistingLockedAdoptionRefusesCandidateAndCaptureChanges(t *testing.T) {
	for _, kind := range []string{"empty", "new-file", "changed-file", "duplicate", "profile", "name", "occurrence", "allocation", "missing-donor", "occupied", "missing-kotlin", "modified-peer", "v3", "same-owner"} {
		t.Run(kind, func(t *testing.T) {
			root, u := existingLockedFixture(t)
			switch kind {
			case "empty":
				u.Generated.Files = nil
			case "new-file":
				u.Generated.Files = append(u.Generated.Files, core.GeneratedFile{Path: "New.kt", Content: "// new"})
			case "changed-file":
				u.Generated.Files[0].Content += "// changed"
			case "duplicate":
				u.Generated.Files = append(u.Generated.Files, u.Generated.Files[0])
			case "profile", "name", "occurrence", "allocation":
				state, err := parseNameLockBytes(u.Generated.NameLock)
				if err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "profile":
					state.Profile.Package = "org.example.changed"
				case "name":
					state.Entries[0].Name = "NewAllocation"
				case "occurrence":
					state.Entries[0].Occurrences = append(state.Entries[0].Occurrences, "/synthetic")
					sort.Strings(state.Entries[0].Occurrences)
				case "allocation":
					state.Entries[0].Shape = digest([]byte("new shape"))
				}
				u.Generated.NameLock, err = state.Encode()
				if err != nil {
					t.Fatal(err)
				}
			case "missing-donor":
				u.Previous.Donor = "absent"
			case "same-owner":
				u.Previous.Donor = u.Previous.Owner
			case "occupied":
				put(t, filepath.Join(root, ownerNameLockFile("consumer")), []byte("foreign"))
			case "missing-kotlin":
				if err := os.Remove(filepath.Join(root, u.Previous.Previous.Files[0].Path)); err != nil {
					t.Fatal(err)
				}
			case "modified-peer":
				put(t, filepath.Join(root, ownerNameLockFile("right")), []byte("changed"))
			case "v3":
				root, _ = lockedAdoptionFixture(t)
				u.Previous.Previous.SourceRoot = root
			}
			before := snapshot(t, root)
			for _, check := range []bool{true, false} {
				if _, err := WriteExistingLockedAdoption(root, u, check); err == nil {
					t.Fatal("invalid accepted", kind)
				}
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("failure changed output", kind)
				}
			}
		})
	}
}

func TestExistingLockedAdoptionEveryBoundaryRollback(t *testing.T) {
	root, u := existingLockedFixture(t)
	before := snapshot(t, root)
	for boundary := 0; boundary < 2; boundary++ {
		_, err := writeOwnersBatchWith(map[string]ExistingLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, stage string) error {
			if phase == "apply" {
				var j journal
				if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); err != nil {
					t.Fatal(err)
				}
				if len(j.Operations) != 2 || j.Operations[1].Path != manifestName || j.Operations[0].Path != ownerNameLockFile("consumer") {
					t.Fatal("not metadata only manifest last", j)
				}
			}
			if phase == "apply" && op == boundary {
				return errors.New("synthetic adoption fault")
			}
			return nil
		}, makeExistingLockedAdoptionPlan)
		errorCode(t, err, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("rollback lost bytes/mode/mtime", boundary)
		}
	}
}

func TestExistingLockedAdoptionCompletePeerCASBeforeStage(t *testing.T) {
	for _, kind := range []string{"manifest", "donor", "peer", "kotlin", "destination"} {
		t.Run(kind, func(t *testing.T) {
			root, u := existingLockedFixture(t)
			before := snapshot(t, root)
			_, err := writeOwnersBatchWith(map[string]ExistingLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
				if phase == "prelock" {
					path := manifestName
					switch kind {
					case "donor":
						path = ownerNameLockFile("left")
					case "peer":
						path = ownerNameLockFile("right")
					case "kotlin":
						path = u.Previous.Previous.Files[0].Path
					case "destination":
						path = ownerNameLockFile("consumer")
					}
					payload := []byte("synthetic foreign state")
					if kind == "manifest" {
						payload = append(u.Previous.Previous.Manifest, '\n')
					}
					put(t, filepath.Join(root, path), payload)
					before = snapshot(t, root)
				}
				return nil
			}, makeExistingLockedAdoptionPlan)
			if err == nil {
				t.Fatal("stale accepted", kind)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("overwrote concurrent state", kind)
			}
			if stages, _ := filepath.Glob(filepath.Join(root, stagePrefix+"*")); len(stages) != 0 {
				t.Fatal("staged stale candidate")
			}
		})
	}
}
