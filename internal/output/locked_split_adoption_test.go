package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"openapi-kt/core"
)

func splitLockedFixture(t testing.TB) (string, SplitLockedAdoptionUpdate) {
	t.Helper()
	root, original := lockedAdoptionFixture(t)
	before, err := ReadSplitLockedAdoptionState(root, []string{"left", "right"})
	if err != nil {
		t.Fatal(err)
	}
	generated := map[string]core.GenerationArtifacts{}
	for index, owner := range []string{"left", "right"} {
		state, err := parseNameLockBytes(before.NameLock)
		if err != nil {
			t.Fatal(err)
		}
		retained := state.Entries[:0]
		for _, entry := range state.Entries {
			occurrences := []string{}
			for i, origin := range entry.Occurrences {
				if i%2 == index {
					occurrences = append(occurrences, origin)
				}
			}
			if len(occurrences) > 0 {
				entry.Occurrences = occurrences
				retained = append(retained, entry)
			}
		}
		state.Entries = retained
		payload, err := state.Encode()
		if err != nil {
			t.Fatal(err)
		}
		// Shared files isolate allocation-occurrence partitioning at this adapter.
		// Actual distinct core-generated file subsets are covered by native tests.
		generated[owner] = core.GenerationArtifacts{Files: append([]core.GeneratedFile(nil), original.Generated.Files...), NameLock: payload}
	}
	return root, SplitLockedAdoptionUpdate{Previous: before, Generated: generated}
}

func TestSplitLockedAdoptionMetadataOnlyProjectionAndRawReplay(t *testing.T) {
	root, u := splitLockedFixture(t)
	before := snapshot(t, root)
	next, err := PlanSplitLockedAdoption(u.Previous, u.Generated)
	if err != nil {
		t.Fatal(err)
	}
	p, err := makeSplitLockedAdoptionPlan(u.Previous.Root, u)
	if err != nil || !p.Changed || len(p.Writes) != 2 || len(p.Removals) != 1 || p.Removals[0] != nameLockFile {
		t.Fatal("not metadata only", p, err)
	}
	if changed, err := WriteSplitLockedAdoption(root, u, true); err != nil || !changed || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("readonly projection", changed, err)
	}
	if changed, err := WriteSplitLockedAdoption(root, u, false); err != nil || !changed {
		t.Fatal(changed, err)
	}
	after := snapshot(t, root)
	for path, entry := range before {
		if filepath.Ext(path) == ".kt" || path == ownerNameLockFile("foreign") {
			if !reflect.DeepEqual(entry, after[path]) {
				t.Fatal("changed Kotlin/unowned data", path)
			}
		}
	}
	if !bytes.Equal(next.Manifest, mustRead(t, filepath.Join(root, manifestName))) {
		t.Fatal("reported manifest differs")
	}
	for owner, payload := range next.NameLocks {
		if !bytes.Equal(payload, mustRead(t, filepath.Join(root, ownerNameLockFile(owner)))) {
			t.Fatal("projected payload differs", owner)
		}
	}
	prior, err := ReadSplitLockedAdoptionState(root, u.Previous.OwnerNames)
	if err != nil {
		t.Fatal(err)
	}
	u.Previous = prior
	if changed, err := WriteSplitLockedAdoption(root, u, false); err != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("v4 replay rewrote metadata", changed, err)
	}
}

func TestSplitLockedAdoptionRejectsMissingChangedOrNewCoverage(t *testing.T) {
	for _, kind := range []string{"missing-file", "edited-file", "new-file", "renamed-allocation", "lost-occurrence", "new-occurrence", "profile", "missing-owner"} {
		root, u := splitLockedFixture(t)
		before := snapshot(t, root)
		left := u.Generated["left"]
		switch kind {
		case "missing-file":
			for owner, a := range u.Generated {
				a.Files = a.Files[1:]
				u.Generated[owner] = a
			}
		case "edited-file":
			left.Files[0].Content += "// changed"
			u.Generated["left"] = left
		case "new-file":
			left.Files = append(left.Files, core.GeneratedFile{Path: "Added.kt", Content: "// added"})
			u.Generated["left"] = left
		case "missing-owner":
			delete(u.Generated, "left")
		default:
			state, err := parseNameLockBytes(left.NameLock)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "renamed-allocation":
				state.Entries[0].Name = "OtherAllocatedName"
			case "lost-occurrence":
				state.Entries[0].Occurrences = state.Entries[0].Occurrences[1:]
				if len(state.Entries[0].Occurrences) == 0 {
					state.Entries = state.Entries[1:]
				}
			case "new-occurrence":
				state.Entries[0].Occurrences = append(state.Entries[0].Occurrences, "/synthetic")
				sort.Strings(state.Entries[0].Occurrences)
			case "profile":
				state.Profile.Package = "org.example.changed"
			}
			left.NameLock, err = state.Encode()
			if err != nil {
				t.Fatal(err)
			}
			u.Generated["left"] = left
		}
		if _, err := WriteSplitLockedAdoption(root, u, false); err == nil {
			t.Fatal("invalid split accepted", kind)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("invalid split changed source", kind)
		}
	}
}

func TestSplitLockedAdoptionEveryMetadataBoundaryRollback(t *testing.T) {
	root, u := splitLockedFixture(t)
	before := snapshot(t, root)
	for op := 0; op < 4; op++ {
		_, err := writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, operation int, stage string) error {
			if phase == "apply" {
				var record journal
				if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &record); err != nil {
					t.Fatal(err)
				}
				if record.Operations[len(record.Operations)-1].Path != manifestName {
					t.Fatal("split manifest not last")
				}
				for _, entry := range record.Operations {
					if filepath.Ext(entry.Path) == ".kt" {
						t.Fatal("Kotlin journaled", entry.Path)
					}
				}
			}
			if phase == "apply" && operation == op {
				return errors.New("synthetic split metadata fault")
			}
			return nil
		}, makeSplitLockedAdoptionPlan)
		errorCode(t, err, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("rollback lost original state", op)
		}
	}
}

func TestSplitLockedAdoptionReplanCASAndOccupiedMetadata(t *testing.T) {
	for _, kind := range []string{"manifest", "lock", "Kotlin", "destination"} {
		root, u := splitLockedFixture(t)
		before := snapshot(t, root)
		_, err := writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				switch kind {
				case "manifest":
					put(t, filepath.Join(root, manifestName), append(mustRead(t, filepath.Join(root, manifestName)), '\n'))
				case "lock":
					put(t, filepath.Join(root, nameLockFile), append(mustRead(t, filepath.Join(root, nameLockFile)), '\n'))
				case "Kotlin":
					put(t, filepath.Join(root, u.Previous.Files[0].Path), []byte("// intervening edit"))
				case "destination":
					put(t, filepath.Join(root, ownerNameLockFile("right")), []byte("foreign"))
				}
				before = snapshot(t, root)
			}
			return nil
		}, makeSplitLockedAdoptionPlan)
		if err == nil {
			t.Fatal("intervening source accepted", kind)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("intervening state lost", kind)
		}
	}
}

func TestSplitLockedAdoptionRawUnprojectedPayloadPreserved(t *testing.T) {
	root, u := splitLockedFixture(t)
	raw := append(append([]byte(" \n"), u.Previous.NameLock...), '\n')
	put(t, filepath.Join(root, nameLockFile), raw)
	m, err := readManifestKind(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	hash := digest(raw)
	m.NameLock = &hash
	data, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), data)
	u.Previous, err = ReadSplitLockedAdoptionState(root, u.Previous.OwnerNames)
	if err != nil {
		t.Fatal(err)
	}
	for owner, a := range u.Generated {
		a.NameLock = u.Previous.NameLock
		u.Generated[owner] = a
	}
	if _, err := WriteSplitLockedAdoption(root, u, false); err != nil {
		t.Fatal(err)
	}
	for _, owner := range u.Previous.OwnerNames {
		if !bytes.Equal(raw, mustRead(t, filepath.Join(root, ownerNameLockFile(owner)))) {
			t.Fatal("lost identical raw payload", owner)
		}
	}
	if _, err := os.Stat(filepath.Join(root, nameLockFile)); !os.IsNotExist(err) {
		t.Fatal("anonymous owned lock retained")
	}
}

func TestSplitLockedAdoptionV4RawReplayAndExactMembership(t *testing.T) {
	root, u := splitLockedFixture(t)
	if _, err := WriteSplitLockedAdoption(root, u, false); err != nil {
		t.Fatal(err)
	}
	m, err := readManifestKind(root, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range u.Previous.OwnerNames {
		raw := append(append([]byte(" \n"), mustRead(t, filepath.Join(root, ownerNameLockFile(owner)))...), '\n')
		put(t, filepath.Join(root, ownerNameLockFile(owner)), raw)
		(*m.NameLocks)[owner] = digest(raw)
	}
	data, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), append(data, '\n'))
	u.Previous, err = ReadSplitLockedAdoptionState(root, u.Previous.OwnerNames)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		if changed, err := WriteSplitLockedAdoption(root, u, check); err != nil || changed || !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("raw multiowner replay changed metadata", changed, err)
		}
	}
	// Coverage remains complete through right; v4 cannot silently alter left's
	// already persisted membership even when every Kotlin byte is unchanged.
	left := u.Generated["left"]
	left.Files = left.Files[1:]
	u.Generated["left"] = left
	_, err = WriteSplitLockedAdoption(root, u, false)
	errorCode(t, err, "E_LOCKED_OWNER_ADOPTION_CHANGED")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("v4 membership refusal changed source")
	}
}

func TestSplitLockedAdoptionRetainedRecoveryAndDirectoryIdentity(t *testing.T) {
	t.Run("directory-identity", func(t *testing.T) {
		root, u := splitLockedFixture(t)
		moved := root + "-original"
		if err := os.Rename(root, moved); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(moved) })
		if err := os.Mkdir(root, 0755); err != nil {
			t.Fatal(err)
		}
		for path, entry := range snapshot(t, moved) {
			if entry.Directory {
				if err := os.MkdirAll(filepath.Join(root, path), 0755); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(root, path), []byte(entry.Data))
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, err := WriteSplitLockedAdoption(root, u, check)
			errorCode(t, err, "E_LOCKED_OWNER_ADOPTION_STALE")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("replaced root was modified")
		}
	})
	t.Run("obstructed-rollback", func(t *testing.T) {
		updates := map[string]SplitLockedAdoptionUpdate{}
		before := map[string]map[string]snapshotEntry{}
		for i := 0; i < 2; i++ {
			root, u := splitLockedFixture(t)
			updates[root] = u
			before[root] = snapshot(t, root)
		}
		roots := sortedKeys(updates)
		_, err := writeOwnersBatchWith(updates, false, func(ri int, phase string, op int, _ string) error {
			if phase == "apply" && ri == 1 && op == 3 {
				obstructLifecycleRollback(t, roots[0], nameLockFile)
				return errors.New("synthetic split rollback obstruction")
			}
			return nil
		}, makeSplitLockedAdoptionPlan)
		errorCode(t, err, "E_RECOVERY_REQUIRED")
		requireLifecycleBackup(t, roots[0], nameLockFile, updates[roots[0]].Previous.NameLock)
		if !bytes.Equal(mustRead(t, filepath.Join(roots[0], manifestName)), updates[roots[0]].Previous.Manifest) || !reflect.DeepEqual(before[roots[1]], snapshot(t, roots[1])) {
			t.Fatal("unobstructed restoration skipped")
		}
		retained := snapshot(t, roots[0])
		for _, check := range []bool{true, false} {
			_, err := WriteSplitLockedAdoption(roots[0], updates[roots[0]], check)
			errorCode(t, err, "E_RECOVERY_REQUIRED")
			if !reflect.DeepEqual(retained, snapshot(t, roots[0])) {
				t.Fatal("retained recovery evidence changed")
			}
		}
	})
}
