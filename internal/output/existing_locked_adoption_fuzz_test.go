package output

import (
	"errors"
	"reflect"
	"testing"
)

func FuzzExistingLockedAdoptionSubsetRollback(f *testing.F) {
	f.Add(uint8(0), uint8(1), true)
	f.Add(uint8(1), uint8(255), false)
	f.Fuzz(func(t *testing.T, boundary, partition uint8, full bool) {
		root, u := existingLockedFixture(t)
		state, e := parseNameLockBytes(u.Generated.NameLock)
		if e != nil {
			t.Fatal(e)
		}
		entries := state.Entries[:0]
		for _, entry := range state.Entries {
			origins := []string{}
			for i, origin := range entry.Occurrences {
				if full || partition>>(i%8)&1 != 0 {
					origins = append(origins, origin)
				}
			}
			if len(origins) > 0 {
				entry.Occurrences = origins
				entries = append(entries, entry)
			}
		}
		state.Entries = entries
		u.Generated.NameLock, e = state.Encode()
		if e != nil {
			t.Fatal(e)
		}
		u.Generated.Files = u.Generated.Files[:1+int(partition)%len(u.Generated.Files)]
		before := snapshot(t, root)
		_, e = writeOwnersBatchWith(map[string]ExistingLockedAdoptionUpdate{root: u}, false, func(_ int, phase string, op int, _ string) error {
			if phase == "apply" && op == int(boundary%2) {
				return errors.New("synthetic shared subset fault")
			}
			return nil
		}, makeExistingLockedAdoptionPlan)
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("subset rollback lost raw inventory/attributes")
		}
	})
}
