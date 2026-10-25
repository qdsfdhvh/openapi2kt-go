package output

import (
	"errors"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzRetainedLockedAdoptionRollback(f *testing.F) {
	for _, seed := range []uint8{0, 1, 2, 3, 7, 19, 255} {
		f.Add(seed, seed%2 == 0)
	}
	f.Fuzz(func(t *testing.T, boundary uint8, named bool) {
		root, u := retainedAdoptionFixture(t, []string{"part"})
		if named {
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
		at := int(boundary) % (len(p.Writes) + len(p.Removals) + 1)
		_, e = writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, _ string) error {
			if phase == "apply" && op == at {
				return errors.New("bounded retained adoption failure")
			}
			return nil
		}, makeRetainedLockedAdoptionPlan)
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("model: retained adoption rollback changed original state")
		}
		if changed, e := WriteRetainedLockedAdoption(root, u, false); e != nil || !changed {
			t.Fatal("retry failed", changed, e)
		}
		s, e := ReadRetainedLockedAdoptionState(root, "original", u.Previous.OwnerNames)
		if e != nil {
			t.Fatal(e)
		}
		u.Previous = s
		after := snapshot(t, root)
		if changed, e := WriteRetainedLockedAdoption(root, u, false); e != nil || changed || !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("model: replay changed raw metadata", changed, e)
		}
	})
}
