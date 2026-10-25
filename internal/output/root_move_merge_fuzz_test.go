package output

import (
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func FuzzRootMoveMergeRollback(f *testing.F) {
	for _, seed := range []uint8{0, 1, 2, 3, 7, 11, 19, 255} {
		f.Add(seed, seed%2 == 0, seed%3 == 0, seed%2 == 1)
	}
	f.Fuzz(func(t *testing.T, boundary uint8, selected, reverse, locked bool) {
		version := 2
		if locked {
			version = 4
		}
		src, dst, s, target := mergeMoveFixture(t, version, core.DataClass)
		if reverse {
			a, b := filepath.Join(filepath.Dir(src), "z-source"), filepath.Join(filepath.Dir(dst), "a-target")
			if e := os.Rename(src, a); e != nil {
				t.Fatal(e)
			}
			if e := os.Rename(dst, b); e != nil {
				t.Fatal(e)
			}
			src, dst = a, b
			s, _ = ReadRootMoveState(src)
		}
		var owners []string
		if selected {
			owners = []string{"alpha"}
		}
		var e *core.Diagnostic
		if selected {
			s, e = BindRootMoveOwnerTarget(s, dst, src, owners)
		} else {
			s, e = BindRootMoveTarget(s, dst, src)
		}
		if e != nil {
			t.Fatal(e)
		}
		moved, retained, e := splitRootMoveOwners(s, owners)
		if e != nil {
			t.Fatal(e)
		}
		sourceOps := 1
		for path := range s.Entries {
			if path != manifestName {
				if _, keep := retained.Entries[path]; !keep {
					sourceOps++
				}
			}
		}
		counts := []int{sourceOps, len(moved.Entries)}
		if reverse {
			counts[0], counts[1] = counts[1], counts[0]
		}
		at := int(boundary) % (counts[0] + counts[1])
		ri := 0
		if at >= counts[0] {
			at -= counts[0]
			ri = 1
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		_, e = moveOutputSelection(src, dst, s, owners, false, func(index int, phase string, op int, _ string) error {
			if phase == "apply" && index == ri && op == at {
				return errors.New("bounded merge fault")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("model: failed merge must preserve both inventories")
		}
		if changed, e := moveOutputSelection(src, dst, s, owners, false, nil); e != nil || !changed {
			t.Fatal("model: retry failed", changed, e)
		}
		result, e := ReadRootMoveState(dst)
		if e != nil {
			t.Fatal(e)
		}
		if len(result.Owners) != len(target.Owners)+len(moved.Owners) {
			t.Fatal("model: lost owner union")
		}
		for path := range target.Entries {
			if path != manifestName && !reflect.DeepEqual(b[path], snapshot(t, dst)[path]) {
				t.Fatal("model: changed original target")
			}
		}
	})
}
