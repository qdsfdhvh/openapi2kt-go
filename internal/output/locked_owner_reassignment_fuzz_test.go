package output

import (
	"errors"
	"reflect"
	"testing"
)

func FuzzLockedOwnerReassignmentRollback(f *testing.F) {
	for _, seed := range []uint8{0, 1, 2, 3, 7, 19, 255} {
		f.Add(seed, seed%2 == 0)
	}
	f.Fuzz(func(t *testing.T, boundary uint8, remove bool) {
		names := []string{"left", "right"}
		if remove {
			names = []string{"newowner"}
		}
		root, u := reassignmentFixture(t, names)
		before := snapshot(t, root)
		p, e := makeLockedOwnerReassignmentPlan(u.Previous.Previous.root, u)
		if e != nil {
			t.Fatal(e)
		}
		at := int(boundary) % (len(p.Writes) + len(p.Removals) + 1)
		_, e = writeOwnersBatchWith(map[string]LockedOwnerReassignmentUpdate{root: u}, false, func(_ int, phase string, op int, _ string) error {
			if phase == "apply" && op == at {
				return errors.New("bounded reassignment failure")
			}
			return nil
		}, makeLockedOwnerReassignmentPlan)
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("model: failed metadata action loses original state")
		}
		if changed, e := WriteLockedOwnerReassignment(root, u, false); e != nil || !changed {
			t.Fatal("model: retry failed", changed, e)
		}
		state, e := ReadLockedOwnerReassignmentState(root, names)
		if e != nil {
			t.Fatal(e)
		}
		u.Previous = state
		after := snapshot(t, root)
		if changed, e := WriteLockedOwnerReassignment(root, u, false); e != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("model: replay must preserve exact state", changed, e)
		}
	})
}
