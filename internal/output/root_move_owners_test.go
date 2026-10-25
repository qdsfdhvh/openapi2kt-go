package output

import (
	"encoding/json"
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveSelectedOwnerRetainsPeerSharedBytes(t *testing.T) {
	for _, version := range []int{2, 4} {
		src, dst, state := moveFixture(t, version)
		beforeSrc, beforeDst := snapshot(t, src), snapshot(t, dst)
		state, err := BindRootMoveOwnerTarget(state, dst, src, []string{"alpha"})
		if err != nil {
			t.Fatal(err)
		}
		changed, err := MoveOutputOwners(src, dst, state, []string{"alpha"}, true)
		if err != nil || !changed || !reflect.DeepEqual(beforeSrc, snapshot(t, src)) || !reflect.DeepEqual(beforeDst, snapshot(t, dst)) {
			t.Fatal("readonly owner move", changed, err)
		}
		changed, err = MoveOutputOwners(src, dst, state, []string{"alpha"}, false)
		if err != nil || !changed {
			t.Fatal(changed, err)
		}
		source, err := ReadRootMoveState(src)
		if err != nil {
			t.Fatal(err)
		}
		target, err := ReadRootMoveState(dst)
		if err != nil {
			t.Fatal(err)
		}
		if source.Version != state.Version || target.Version != state.Version || len(source.Owners) != 1 || len(target.Owners) != 1 || !reflect.DeepEqual(source.Owners["peer"], state.Owners["peer"]) || !reflect.DeepEqual(target.Owners["alpha"], state.Owners["alpha"]) {
			t.Fatal("lost selected/retained owner inventory")
		}
		for _, path := range state.Owners["alpha"] {
			if !reflect.DeepEqual(beforeSrc[filepath.FromSlash(path)], snapshot(t, dst)[filepath.FromSlash(path)]) || !reflect.DeepEqual(beforeSrc[filepath.FromSlash(path)], snapshot(t, src)[filepath.FromSlash(path)]) {
				t.Fatal("lost shared Kotlin bytes/attributes", path)
			}
		}
		for owner, payload := range state.NameLocks {
			actual := source.NameLocks[owner]
			if owner == "alpha" {
				actual = target.NameLocks[owner]
			}
			if !reflect.DeepEqual(payload, actual) {
				t.Fatal("rewrote raw owner lock", owner)
			}
		}
	}
}

func TestRootMoveSelectedOwnerEveryJournalBoundaryRollback(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	selected, retained, err := splitRootMoveOwners(s, []string{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	operations := []int{1, len(selected.Entries)}
	for path := range s.Entries {
		if path != manifestName {
			if _, keep := retained.Entries[path]; !keep {
				operations[0]++
			}
		}
	}
	a, b := snapshot(t, src), snapshot(t, dst)
	for index, count := range operations {
		for op := 0; op < count; op++ {
			_, err := moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(root int, phase string, operation int, stage string) error {
				if phase == "apply" {
					var record journal
					if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &record); err != nil {
						t.Fatal(err)
					}
					if record.Operations[len(record.Operations)-1].Path != manifestName {
						t.Fatal("partial manifest was not last")
					}
				}
				if root == index && phase == "apply" && operation == op {
					return errors.New("synthetic selected-owner failure")
				}
				return nil
			})
			errorCode(t, err, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("partial rollback lost original bytes/attributes", index, op)
			}
		}
		_, err := moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(root int, phase string, _ int, _ string) error {
			if root == index && phase == "prepare" {
				return errors.New("synthetic partial prepare failure")
			}
			return nil
		})
		errorCode(t, err, "E_OUTPUT_IO")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("partial prepare changed roots", index)
		}
	}
}

func TestRootMoveSelectedOwnerCASAndForeignTarget(t *testing.T) {
	for _, kind := range []string{"peer-manifest", "target-manifest", "peer-code"} {
		src, dst, s := moveFixture(t, 4)
		s, err := BindRootMoveOwnerTarget(s, dst, src, []string{"alpha"})
		if err != nil {
			t.Fatal(err)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		_, err = moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				switch kind {
				case "peer-manifest":
					put(t, filepath.Join(src, manifestName), append(mustRead(t, filepath.Join(src, manifestName)), '\n'))
				case "target-manifest":
					put(t, filepath.Join(dst, manifestName), s.Manifest)
				case "peer-code":
					put(t, filepath.Join(src, s.Owners["peer"][0]), []byte("// intervening edit"))
				}
				a, b = snapshot(t, src), snapshot(t, dst)
			}
			return nil
		})
		code := "E_OUTPUT_ROOT_MOVE_STALE"
		if kind == "target-manifest" {
			code = "E_OUTPUT_ROOT_MOVE_OCCUPIED"
		}
		if kind == "peer-code" {
			code = "E_OUTPUT_MODIFIED"
		}
		errorCode(t, err, code)
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("partial move lost intervening state", kind)
		}
	}
}

func TestRootMoveSelectedOwnerExclusiveFilesAndMixedLockModes(t *testing.T) {
	for _, moveLocked := range []bool{false, true} {
		base := t.TempDir()
		src, dst := filepath.Join(base, "source"), filepath.Join(base, "target")
		a := singleGenerated(t, singleLockRequest(t, core.DataClass, moveLocked))
		b := singleGenerated(t, singleLockRequest(t, core.DataClass, !moveLocked))
		a.Files = append(a.Files, core.GeneratedFile{Path: "org/example/OnlyAlpha.kt", Content: "// synthetic alpha owned file\n"})
		b.Files = append(b.Files, core.GeneratedFile{Path: "org/example/OnlyPeer.kt", Content: "// synthetic peer owned file\n"})
		previous := map[string][]byte{}
		if moveLocked {
			previous["alpha"] = nil
		} else {
			previous["peer"] = nil
		}
		_, err := WriteNameLockedOwners(src, lockedUpdate([]string{"alpha", "peer"}, map[string]core.GenerationArtifacts{"alpha": a, "peer": b}, previous, true), false)
		if err != nil {
			t.Fatal(err)
		}
		state, err := ReadRootMoveState(src)
		if err != nil {
			t.Fatal(err)
		}
		before := snapshot(t, src)
		changed, err := MoveOutputOwners(src, dst, state, []string{"alpha"}, false)
		if err != nil || !changed {
			t.Fatal(changed, err)
		}
		source, err := ReadRootMoveState(src)
		if err != nil {
			t.Fatal(err)
		}
		target, err := ReadRootMoveState(dst)
		if err != nil {
			t.Fatal(err)
		}
		if source.Version != 4 || target.Version != 4 || (len(target.NameLocks) == 1) != moveLocked || (len(source.NameLocks) == 1) == moveLocked {
			t.Fatal("lost mixed locked/unlocked modes")
		}
		if _, err := os.Lstat(filepath.Join(src, "org/example/OnlyAlpha.kt")); !os.IsNotExist(err) {
			t.Fatal("exclusive moved source file retained")
		}
		if _, err := os.Lstat(filepath.Join(dst, "org/example/OnlyPeer.kt")); !os.IsNotExist(err) {
			t.Fatal("unselected peer copied")
		}
		for _, path := range state.Owners["alpha"] {
			if !reflect.DeepEqual(before[path], snapshot(t, dst)[path]) {
				t.Fatal("lost selected payload", path)
			}
		}
		for _, path := range state.Owners["peer"] {
			if !reflect.DeepEqual(before[path], snapshot(t, src)[path]) {
				t.Fatal("changed retained payload", path)
			}
		}
	}
}

func TestRootMoveOwnerSelectionRefusesInvalidInventory(t *testing.T) {
	for _, version := range []int{1, 2, 3, 4} {
		for _, owners := range [][]string{{}, {"missing"}, {"alpha", "alpha"}} {
			src, dst, state := moveFixture(t, version)
			a, b := snapshot(t, src), snapshot(t, dst)
			if _, err := MoveOutputOwners(src, dst, state, owners, false); err == nil {
				t.Fatal("invalid owner selection accepted", version, owners)
			}
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("invalid selection wrote output")
			}
		}
	}
}
