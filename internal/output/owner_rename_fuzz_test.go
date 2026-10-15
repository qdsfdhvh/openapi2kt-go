package output

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzOwnerRenameTransitions(f *testing.F) {
	for _, seed := range [][]byte{{0, 1, 2, 3, 4}, {1, 0, 0, 0}, {1, 4, 3, 2, 1, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, steps []byte) {
		if len(steps) == 0 || len(steps) > 8 {
			t.Skip()
		}
		root, a, _ := renameFixture(t, steps[0]%2 == 1)
		initial := snapshot(t, root)
		current := "old"
		for _, step := range steps {
			state, e := ReadOwnerRenameState(root)
			if e != nil {
				t.Fatal(e)
			}
			next := "new"
			if current == "new" {
				next = "old"
			}
			u := OwnerRenameUpdate{Previous: state, Renames: map[string]string{next: current}, Generated: map[string]core.GenerationArtifacts{next: a}}
			before := snapshot(t, root)
			switch step % 5 {
			case 0:
				changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, true)
				if e != nil || !changed {
					t.Fatal(changed, e)
				}
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("fuzz check changed state")
				}
				if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil || !changed {
					t.Fatal(changed, e)
				}
				current = next
			case 1:
				u.Renames = map[string]string{current: current}
				u.Generated = map[string]core.GenerationArtifacts{current: a}
				if changed, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false); e != nil || changed {
					t.Fatal(changed, e)
				}
			case 2:
				candidate := a
				candidate.Files = append([]core.GeneratedFile(nil), a.Files...)
				candidate.Files[0].Content += "// changed"
				u.Generated[next] = candidate
				_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false)
				errorCode(t, e, "E_OWNER_RENAME_CHANGED")
			case 3:
				u.Previous.Manifest = append(append([]byte(nil), u.Previous.Manifest...), ' ')
				_, e := WriteOwnerRenamesBatch(map[string]OwnerRenameUpdate{root: u}, false)
				errorCode(t, e, "E_OWNER_RENAME_STALE")
			case 4:
				boundary := 0
				if a.NameLock != nil {
					boundary = int(step/5) % 3
				}
				_, e := writeOwnersBatchWith(map[string]OwnerRenameUpdate{root: u}, false, func(_ int, phase string, index int, _ string) error {
					if phase == "apply" && index == boundary {
						return errors.New("synthetic fuzz failure")
					}
					return nil
				}, makeOwnerRenamePlan)
				errorCode(t, e, "E_OUTPUT_COMMIT")
			}
			after := snapshot(t, root)
			if step%5 != 0 && !reflect.DeepEqual(before, after) {
				t.Fatal("no-op/failure changed state")
			}
			for _, file := range a.Files {
				if !reflect.DeepEqual(initial[file.Path], after[file.Path]) {
					t.Fatal("source/shared changed")
				}
			}
			m, e := readManifestKind(root, true, true)
			if e != nil {
				t.Fatal(e)
			}
			if len(*m.Owners) != 2 || len((*m.Owners)[current]) != len(a.Files) || !reflect.DeepEqual((*m.Owners)[current], (*m.Owners)["peer"]) {
				t.Fatal("reference owner model", current, m)
			}
			if a.NameLock != nil && string(mustRead(t, filepath.Join(root, ownerNameLockFile(current)))) != string(a.NameLock) {
				t.Fatal("state allocations changed")
			}
		}
	})
}
