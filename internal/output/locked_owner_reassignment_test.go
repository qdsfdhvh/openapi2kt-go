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
	"time"

	"openapi-kt/core"
	"openapi-kt/internal/schema"
)

func reassignmentFixture(t testing.TB, names []string) (string, LockedOwnerReassignmentUpdate) {
	t.Helper()
	root, u := splitLockedFixture(t)
	if _, e := WriteSplitLockedAdoption(root, u, false); e != nil {
		t.Fatal(e)
	}
	// The adoption fixture includes a deliberately foreign metadata sentinel.
	// A reassignment requires a fully owned metadata boundary; test it separately.
	if e := os.Remove(filepath.Join(root, ownerNameLockFile("foreign"))); e != nil {
		t.Fatal(e)
	}
	m, e := readManifestKind(root, true, true)
	if e != nil {
		t.Fatal(e)
	}
	for _, owner := range sortedKeys(*m.NameLocks) {
		raw := append([]byte(" \n"), mustRead(t, filepath.Join(root, ownerNameLockFile(owner)))...)
		raw = append(raw, '\n')
		put(t, filepath.Join(root, ownerNameLockFile(owner)), raw)
		(*m.NameLocks)[owner] = digest(raw)
	}
	raw, _ := json.MarshalIndent(m, "", "  ")
	put(t, filepath.Join(root, manifestName), append(raw, '\n'))
	for i, path := range sortedKeys(snapshot(t, root)) {
		if path == lockName {
			continue
		}
		full := filepath.Join(root, path)
		info, err := os.Stat(full)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			continue
		}
		mode := os.FileMode(0640)
		if i%2 == 1 {
			mode = 0600
		}
		if e := os.Chmod(full, mode); e != nil {
			t.Fatal(e)
		}
		stamp := time.Unix(1200000000+int64(i), 0)
		if e := os.Chtimes(full, stamp, stamp); e != nil {
			t.Fatal(e)
		}
	}
	state, e := ReadLockedOwnerReassignmentState(root, names)
	if e != nil {
		t.Fatal(e)
	}
	original, e := parseNameLockBytes(state.NameLock)
	if e != nil {
		t.Fatal(e)
	}
	empty, e := (&schema.HoistedNameLock{Version: 1, Profile: original.Profile, Entries: []schema.NameLockEntry{}}).Encode()
	if e != nil {
		t.Fatal(e)
	}
	generated := map[string]core.GenerationArtifacts{}
	for _, name := range names {
		generated[name] = core.GenerationArtifacts{Files: append([]core.GeneratedFile{}, state.Previous.Files...), NameLock: state.NameLock}
	}
	if _, exists := generated["left"]; exists && len(names) > 1 {
		g := generated["left"]
		g.Files = g.Files[:1]
		g.NameLock = empty
		generated["left"] = g
	}
	return root, LockedOwnerReassignmentUpdate{Previous: state, Generated: generated}
}

func TestLockedOwnerReassignmentMetadataOnlyCoverageRawDonorsAndReplay(t *testing.T) {
	for _, names := range [][]string{{"left", "right"}, {"right"}, {"left", "newowner"}, {"newowner"}} {
		root, u := reassignmentFixture(t, names)
		before := snapshot(t, root)
		p, e := makeLockedOwnerReassignmentPlan(u.Previous.Previous.root, u)
		if e != nil || !p.Changed {
			t.Fatal(p, e)
		}
		for path := range p.Writes {
			if filepath.Ext(path) == ".kt" {
				t.Fatal("Kotlin write in metadata action")
			}
		}
		for _, path := range p.Removals {
			if filepath.Ext(path) == ".kt" {
				t.Fatal("Kotlin removal in metadata action")
			}
		}
		if changed, e := WriteLockedOwnerReassignment(root, u, true); e != nil || !changed || !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("readonly reassignment", changed, e)
		}
		if changed, e := WriteLockedOwnerReassignment(root, u, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		after := snapshot(t, root)
		for path, entry := range before {
			if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(entry, after[path]) {
				t.Fatal("Kotlin/handwritten bytes or attributes changed", path)
			}
		}
		state, e := ReadLockedOwnerReassignmentState(root, names)
		if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(sortedKeys(state.Previous.Owners), state.OwnerNames) || !bytes.Equal(state.NameLock, u.Previous.NameLock) {
			t.Fatal("global owner/allocation coverage lost")
		}
		for _, old := range sortedKeys(u.Previous.Previous.Owners) {
			if _, keep := u.Generated[old]; !keep {
				if _, err := os.Lstat(filepath.Join(root, ownerNameLockFile(old))); !os.IsNotExist(err) {
					t.Fatal("removed owner raw lock remains")
				}
			}
		}
		u.Previous = state
		for _, check := range []bool{true, false} {
			if changed, e := WriteLockedOwnerReassignment(root, u, check); e != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
				t.Fatal("exact reassignment replay changed raw metadata", changed, e)
			}
		}
	}
	// Add a shared owner using an unchanged semantic donor. The new file must
	// preserve the donor's noncanonical raw payload, mode and mtime.
	root, u := reassignmentFixture(t, []string{"extra", "left", "right"})
	for _, name := range []string{"left", "right"} {
		g := u.Generated[name]
		g.Files = append([]core.GeneratedFile{}, u.Previous.Previous.Files...)
		g.NameLock = u.Previous.Previous.NameLocks[name]
		u.Generated[name] = g
	}
	g := u.Generated["extra"]
	g.NameLock = u.Previous.Previous.NameLocks["left"]
	u.Generated["extra"] = g
	before := snapshot(t, root)
	next, e := PlanLockedOwnerReassignment(u.Previous, u.Generated)
	if e != nil || next.Donors["extra"] != "left" {
		t.Fatal("lost original raw donor authority", e)
	}
	if changed, e := WriteLockedOwnerReassignment(root, u, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	after := snapshot(t, root)
	if !reflect.DeepEqual(before[ownerNameLockFile("left")], after[ownerNameLockFile("extra")]) {
		t.Fatal("new shared donor raw bytes/attrs changed")
	}
	for _, name := range []string{"left", "right"} {
		if !reflect.DeepEqual(before[ownerNameLockFile(name)], after[ownerNameLockFile(name)]) {
			t.Fatal("unchanged owner raw state rewritten")
		}
	}
}

func TestLockedOwnerReassignmentEveryMetadataJournalBoundary(t *testing.T) {
	root, u := reassignmentFixture(t, []string{"newowner"})
	before := snapshot(t, root)
	p, e := makeLockedOwnerReassignmentPlan(u.Previous.Previous.root, u)
	if e != nil {
		t.Fatal(e)
	}
	count := len(p.Writes) + len(p.Removals) + 1
	for op := 0; op < count; op++ {
		_, e := writeOwnersBatchWith(map[string]LockedOwnerReassignmentUpdate{root: u}, false, func(_ int, phase string, index int, stage string) error {
			if phase == "apply" {
				var j journal
				if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
					t.Fatal(e)
				}
				if j.Operations[len(j.Operations)-1].Path != manifestName {
					t.Fatal("manifest not last")
				}
				for _, entry := range j.Operations {
					if filepath.Ext(entry.Path) == ".kt" {
						t.Fatal("Kotlin staged for metadata reassignment")
					}
				}
				if index == op {
					return errors.New("synthetic reassignment boundary")
				}
			}
			return nil
		}, makeLockedOwnerReassignmentPlan)
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("reassignment rollback lost bytes or attrs", op)
		}
	}
	_, e = writeOwnersBatchWith(map[string]LockedOwnerReassignmentUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prepare" {
			return errors.New("synthetic reassignment prepare failure")
		}
		return nil
	}, makeLockedOwnerReassignmentPlan)
	errorCode(t, e, "E_OUTPUT_IO")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("reassignment prepare wrote")
	}
}

func TestLockedOwnerReassignmentRejectsChangedCandidatesAndAuthority(t *testing.T) {
	for _, kind := range []string{"code", "missing-code", "new-code", "missing-origin", "new-origin", "allocation-name", "profile", "source-lock", "anchor", "owner-name", "duplicate-file", "empty-lock"} {
		root, u := reassignmentFixture(t, []string{"left", "right"})
		g := u.Generated["right"]
		switch kind {
		case "code":
			g.Files[0].Content += "// change"
		case "missing-code":
			g.Files = g.Files[:1]
		case "new-code":
			g.Files = append(g.Files, core.GeneratedFile{Path: "other/New.kt", Content: "// new"})
		case "source-lock":
			u.Previous.Previous.NameLocks["left"] = []byte("mutated")
		case "anchor":
			u.Previous.NameLock = append(u.Previous.NameLock, ' ')
		case "owner-name":
			u.Previous.OwnerNames = []string{"left", "unknown"}
		case "duplicate-file":
			g.Files = append(g.Files, g.Files[0])
		case "empty-lock":
			g.NameLock = nil
		default:
			state, e := parseNameLockBytes(g.NameLock)
			if e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "missing-origin":
				state.Entries[0].Occurrences = state.Entries[0].Occurrences[:1]
			case "new-origin":
				state.Entries[0].Occurrences = append(state.Entries[0].Occurrences, "/synthetic/new")
				sort.Strings(state.Entries[0].Occurrences)
			case "allocation-name":
				state.Entries[0].Name += "Changed"
			case "profile":
				state.Profile.Package = "org.example.other"
			}
			g.NameLock, e = state.Encode()
			if e != nil {
				t.Fatal(e)
			}
		}
		u.Generated["right"] = g
		before := snapshot(t, root)
		if _, e := WriteLockedOwnerReassignment(root, u, false); e == nil {
			t.Fatal("invalid candidate accepted", kind)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("invalid reassignment wrote", kind)
		}
	}
}

func TestLifecycleRootReaderCumulativeActualKotlinBudget(t *testing.T) {
	root := t.TempDir()
	a := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "A.kt", Content: "1234"}, {Path: "B.kt", Content: "567"}}}
	if _, e := Write(root, a, false); e != nil {
		t.Fatal(e)
	}
	if _, e := readRootMoveState(root, 7); e != nil {
		t.Fatal("exact source budget rejected", e)
	}
	if _, e := readRootMoveState(root, 6); e == nil || e.Code != "E_OUTPUT_LIMIT" {
		t.Fatal("actual cumulative source budget not enforced", e)
	}
	if _, e := ReadRootMoveState(root); e != nil {
		t.Fatal("ordinary root move cap changed", e)
	}
}
