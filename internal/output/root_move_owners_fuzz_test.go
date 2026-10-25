package output

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func FuzzRootMoveOwnersRollback(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(9), true, true)
	f.Fuzz(func(t *testing.T, boundary uint8, locked, reverse bool) {
		version := 2
		if locked {
			version = 4
		}
		src, dst, s := moveFixture(t, version)
		if reverse {
			from, to := filepath.Join(filepath.Dir(src), "z-source"), filepath.Join(filepath.Dir(src), "a-target")
			if err := os.Rename(src, from); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(dst, to); err != nil {
				t.Fatal(err)
			}
			src, dst = from, to
		}
		state, err := ReadRootMoveState(src)
		if err != nil {
			t.Fatal(err)
		}
		s = state
		selected, retained, err := splitRootMoveOwners(s, []string{"alpha"})
		if err != nil {
			t.Fatal(err)
		}
		operations := map[string]int{src: 1, dst: len(selected.Entries)}
		for path := range s.Entries {
			if path != manifestName {
				if _, keep := retained.Entries[path]; !keep {
					operations[src]++
				}
			}
		}
		roots := sortedKeys(operations)
		index := int(boundary) % 2
		op := int(boundary/2) % operations[roots[index]]
		a, b := snapshot(t, src), snapshot(t, dst)
		_, err = moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(root int, phase string, operation int, _ string) error {
			if root == index && phase == "apply" && operation == op {
				return errors.New("synthetic selected-owner fuzz fault")
			}
			return nil
		})
		errorCode(t, err, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("owner split rollback lost state")
		}
	})
}
