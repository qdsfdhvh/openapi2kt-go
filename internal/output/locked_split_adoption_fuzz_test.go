package output

import (
	"errors"
	"reflect"
	"testing"
)

func FuzzSplitLockedAdoptionRollback(f *testing.F) {
	f.Add(uint8(0), uint8(0), false)
	f.Add(uint8(3), uint8(255), true)
	f.Fuzz(func(t *testing.T, boundary, partition uint8, shared bool) {
		root, u := splitLockedFixture(t)
		// Redistribute each original occurrence across both projected states,
		// preserving exact union while varying one-sided and shared allocations.
		for index, owner := range u.Previous.OwnerNames {
			state, err := parseNameLockBytes(u.Previous.NameLock)
			if err != nil {
				t.Fatal(err)
			}
			entries := state.Entries[:0]
			for _, entry := range state.Entries {
				origins := []string{}
				for i, origin := range entry.Occurrences {
					if shared || int(partition>>(i%8)&1) == index {
						origins = append(origins, origin)
					}
				}
				if len(origins) > 0 {
					entry.Occurrences = origins
					entries = append(entries, entry)
				}
			}
			state.Entries = entries
			candidate := u.Generated[owner]
			candidate.NameLock, err = state.Encode()
			if err != nil {
				t.Fatal(err)
			}
			u.Generated[owner] = candidate
		}
		before := snapshot(t, root)
		_, err := writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, _ string) error {
			if phase == "apply" && op == int(boundary%4) {
				return errors.New("synthetic projected partition fault")
			}
			return nil
		}, makeSplitLockedAdoptionPlan)
		errorCode(t, err, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("projected partition rollback lost original bytes/attributes")
		}
	})
}
