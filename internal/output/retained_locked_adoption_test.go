package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/core"
	"openapi-kt/internal/schema"
)

func retainedAdoptionFixture(t testing.TB, names []string) (string, RetainedLockedAdoptionUpdate) {
	t.Helper()
	root, _ := lockedAdoptionFixture(t)
	// Remove only this isolated fixture's deliberate foreign sentinel, then test
	// the stronger retained-adoption foreign-metadata boundary independently.
	if e := os.Remove(filepath.Join(root, ownerNameLockFile("foreign"))); e != nil {
		t.Fatal(e)
	}
	raw := append([]byte(" \n"), mustRead(t, filepath.Join(root, nameLockFile))...)
	raw = append(raw, '\n')
	put(t, filepath.Join(root, nameLockFile), raw)
	m, e := readManifestKind(root, false, true)
	if e != nil {
		t.Fatal(e)
	}
	hash := digest(raw)
	m.NameLock = &hash
	data, _ := json.MarshalIndent(m, "", "  ")
	put(t, filepath.Join(root, manifestName), append(data, '\n'))
	for _, path := range []string{nameLockFile, manifestName} {
		full := filepath.Join(root, path)
		if e := os.Chmod(full, 0640); e != nil {
			t.Fatal(e)
		}
		stamp := time.Unix(1234567890, 0)
		if e := os.Chtimes(full, stamp, stamp); e != nil {
			t.Fatal(e)
		}
	}
	s, e := ReadRetainedLockedAdoptionState(root, "original", names)
	if e != nil {
		t.Fatal(e)
	}
	source, e := parseNameLockBytes(s.NameLock)
	if e != nil {
		t.Fatal(e)
	}
	generated := map[string]core.GenerationArtifacts{}
	for i, name := range names {
		state := &schema.HoistedNameLock{Version: 1, Profile: source.Profile, Entries: []schema.NameLockEntry{}}
		if i%2 == 0 {
			entry := source.Entries[0]
			entry.Occurrences = append([]string{}, entry.Occurrences[:1]...)
			state.Entries = append(state.Entries, entry)
		}
		payload, e := state.Encode()
		if e != nil {
			t.Fatal(e)
		}
		generated[name] = core.GenerationArtifacts{Files: append([]core.GeneratedFile{}, s.Previous.Files[:1]...), NameLock: payload}
	}
	return root, RetainedLockedAdoptionUpdate{Previous: s, Generated: generated}
}

func TestRetainedLockedAdoptionActualSourceOwnershipRawAttrsAndReplay(t *testing.T) {
	for _, names := range [][]string{{"part"}, {"part", "other"}} {
		root, u := retainedAdoptionFixture(t, names)
		before := snapshot(t, root)
		next, e := PlanRetainedLockedAdoption(u.Previous, u.Generated)
		if e != nil {
			t.Fatal(e)
		}
		p, e := makeRetainedLockedAdoptionPlan(u.Previous.Previous.root, u)
		if e != nil || !p.Changed || !reflect.DeepEqual(p.Removals, []string{nameLockFile}) {
			t.Fatal("invalid owned metadata plan", p, e)
		}
		for path := range p.Writes {
			if filepath.Ext(path) == ".kt" {
				t.Fatal("Kotlin staged for partial adoption")
			}
		}
		if changed, e := WriteRetainedLockedAdoption(root, u, true); e != nil || !changed || !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("readonly changed original", changed, e)
		}
		if changed, e := WriteRetainedLockedAdoption(root, u, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		after := snapshot(t, root)
		for path, entry := range before {
			if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(entry, after[path]) {
				t.Fatal("original Kotlin/attrs changed", path)
			}
		}
		if !reflect.DeepEqual(before[nameLockFile], after[ownerNameLockFile("original")]) || !bytes.Equal(next.NameLocks["original"], u.Previous.NameLock) {
			t.Fatal("direct raw lock/attrs not transferred verbatim")
		}
		if _, e := os.Lstat(filepath.Join(root, nameLockFile)); !os.IsNotExist(e) {
			t.Fatal("owned direct slot survived v4 adoption")
		}
		var manifest struct{ Owners map[string][]string }
		if e := json.Unmarshal(mustRead(t, filepath.Join(root, manifestName)), &manifest); e != nil {
			t.Fatal(e)
		}
		if len(manifest.Owners["original"]) != len(u.Previous.Previous.Files) {
			t.Fatal("original full source ownership discarded")
		}
		s, e := ReadRetainedLockedAdoptionState(root, "original", names)
		if e != nil {
			t.Fatal(e)
		}
		u.Previous = s
		for _, check := range []bool{true, false} {
			if changed, e := WriteRetainedLockedAdoption(root, u, check); e != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
				t.Fatal("exact replay changed raw metadata", changed, e)
			}
		}
		// Add another actual subset to v4 without regenerating existing candidates.
		s, e = ReadRetainedLockedAdoptionState(root, "original", []string{"extra"})
		if e != nil {
			t.Fatal(e)
		}
		g := u.Generated[names[0]]
		added := RetainedLockedAdoptionUpdate{Previous: s, Generated: map[string]core.GenerationArtifacts{"extra": g}}
		if changed, e := WriteRetainedLockedAdoption(root, added, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		addedAfter := snapshot(t, root)
		for path := range s.Previous.Entries {
			if path != manifestName && !reflect.DeepEqual(after[path], addedAfter[path]) {
				t.Fatal("v4 addition rewrote original peer payload", path)
			}
		}
	}
}

func TestRetainedLockedAdoptionEveryMetadataJournalBoundary(t *testing.T) {
	for _, v4 := range []bool{false, true} {
		root, u := retainedAdoptionFixture(t, []string{"part"})
		if v4 {
			if _, e := WriteRetainedLockedAdoption(root, u, false); e != nil {
				t.Fatal(e)
			}
			s, e := ReadRetainedLockedAdoptionState(root, "original", []string{"extra"})
			if e != nil {
				t.Fatal(e)
			}
			u = RetainedLockedAdoptionUpdate{Previous: s, Generated: map[string]core.GenerationArtifacts{"extra": u.Generated["part"]}}
		}
		before := snapshot(t, root)
		p, e := makeRetainedLockedAdoptionPlan(u.Previous.Previous.root, u)
		if e != nil {
			t.Fatal(e)
		}
		count := len(p.Writes) + len(p.Removals) + 1
		for op := 0; op < count; op++ {
			_, e := writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, index int, stage string) error {
				if phase == "apply" {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if j.Operations[len(j.Operations)-1].Path != manifestName {
						t.Fatal("retained manifest not last")
					}
					for _, entry := range j.Operations {
						if filepath.Ext(entry.Path) == ".kt" {
							t.Fatal("Kotlin journal in metadata-only action")
						}
					}
					if index == op {
						return errors.New("synthetic retained adoption fault")
					}
				}
				return nil
			}, makeRetainedLockedAdoptionPlan)
			errorCode(t, e, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("partial adoption rollback lost old bytes/attrs", v4, op)
			}
		}
		_, e = writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prepare" {
				return errors.New("synthetic retained prepare")
			}
			return nil
		}, makeRetainedLockedAdoptionPlan)
		errorCode(t, e, "E_OUTPUT_IO")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("retained prepare wrote")
		}
	}
}

func TestRetainedLockedAdoptionRefusesInvalidCandidatesAndActualSource(t *testing.T) {
	for _, kind := range []string{"code", "new-code", "duplicate", "new-origin", "name", "profile", "missing-lock", "source-raw", "source-manifest", "source-entry", "retainer-collision", "retainer-source", "candidate-names"} {
		root, u := retainedAdoptionFixture(t, []string{"part"})
		g := u.Generated["part"]
		switch kind {
		case "code":
			g.Files[0].Content += "// change"
		case "new-code":
			g.Files[0].Path = "new/Unowned.kt"
		case "duplicate":
			g.Files = append(g.Files, g.Files[0])
		case "missing-lock":
			g.NameLock = nil
		case "source-raw":
			u.Previous.NameLock = append(u.Previous.NameLock, ' ')
		case "source-manifest":
			u.Previous.Previous.Manifest = append(u.Previous.Previous.Manifest, ' ')
		case "source-entry":
			delete(u.Previous.Previous.Entries, g.Files[0].Path)
		case "retainer-collision":
			u.Previous.retained = "part"
		case "retainer-source":
			u.Previous.Previous.NameLocks[""] = []byte("foreign")
		case "candidate-names":
			u.Previous.OwnerNames = []string{"unknown"}
		default:
			state, e := parseNameLockBytes(g.NameLock)
			if e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "new-origin":
				state.Entries[0].Occurrences = []string{"/new/origin"}
			case "name":
				state.Entries[0].Name += "Other"
			case "profile":
				state.Profile.Package = "org.example.other"
			}
			g.NameLock, e = state.Encode()
			if e != nil {
				t.Fatal(e)
			}
		}
		u.Generated["part"] = g
		before := snapshot(t, root)
		if _, e := WriteRetainedLockedAdoption(root, u, false); e == nil {
			t.Fatal("invalid retained candidate/source accepted", kind)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("invalid retained action wrote", kind)
		}
	}
}
