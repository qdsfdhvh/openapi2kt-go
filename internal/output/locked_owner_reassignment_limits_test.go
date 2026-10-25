package output

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/schema"
)

func TestLockedOwnerReassignmentOriginalUnionExactLimits(t *testing.T) {
	for _, kind := range []string{"entries", "occurrences"} {
		for _, over := range []bool{false, true} {
			root := t.TempDir()
			a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
			profile, e := parseNameLockBytes(a.NameLock)
			if e != nil {
				t.Fatal(e)
			}
			generated := map[string]core.GenerationArtifacts{}
			previous := map[string][]byte{}
			names := []string{"one", "two"}
			if over {
				names = append(names, "three")
			}
			for side, owner := range names {
				state := &schema.HoistedNameLock{Version: 1, Profile: profile.Profile, Entries: []schema.NameLockEntry{}}
				count := schema.MaxNameLockEntries / 2
				if kind == "occurrences" {
					count = 1
				}
				if side == 2 {
					count = 1
				}
				for j := 0; j < count; j++ {
					id := side*count + j
					if side == 2 {
						id = 999999
					}
					entry := schema.NameLockEntry{Role: "ResponseDto", Shape: digest([]byte(fmt.Sprintf("synthetic-%d", id))), Name: fmt.Sprintf("X%d", id), Occurrences: []string{fmt.Sprintf("/source/%d", id)}}
					if kind == "occurrences" {
						entry.Occurrences = nil
						n := schema.MaxNameLockOccurrences / 2
						if side == 2 {
							n = 1
						}
						for k := 0; k < n; k++ {
							entry.Occurrences = append(entry.Occurrences, fmt.Sprintf("/source/%d/%05d", side, k))
						}
					}
					state.Entries = append(state.Entries, entry)
				}
				payload, e := state.Encode()
				if e != nil {
					t.Fatal("bad bounded source fixture", kind, e)
				}
				g := a
				g.NameLock = payload
				generated[owner] = g
				previous[owner] = nil
			}
			if _, e := WriteNameLockedOwners(root, lockedUpdate(names, generated, previous, true), false); e != nil {
				t.Fatal(e)
			}
			before := snapshot(t, root)
			_, e = ReadLockedOwnerReassignmentState(root, []string{"desired"})
			if over {
				errorCode(t, e, "E_NAME_LOCK_LIMIT")
			} else if e != nil {
				t.Fatal("exact original union limit refused", kind, e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("source union limit observation wrote")
			}
		}
	}
}

func TestLockedOwnerReassignmentDesiredOwnerCountBoundary(t *testing.T) {
	root, u := reassignmentFixture(t, []string{"left", "right"})
	before := snapshot(t, root)
	names := []string{}
	for i := 0; i < maxLockedOwners; i++ {
		names = append(names, fmt.Sprintf("owner%d", i))
	}
	if _, e := ReadLockedOwnerReassignmentState(root, names); e != nil {
		t.Fatal("exact desired owner count refused", e)
	}
	if _, e := ReadLockedOwnerReassignmentState(root, append(names, "oneextra")); e == nil {
		t.Fatal("desired owner count exceeded original cap")
	}
	for _, names := range [][]string{nil, {"left", "left"}, {"../invalid"}} {
		if _, e := ReadLockedOwnerReassignmentState(root, names); e == nil {
			t.Fatal("invalid owner list accepted", names)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("owner limit observation wrote")
	}
	// A different raw payload with the same semantic state is not new authority.
	u.Previous.NameLock = append(u.Previous.NameLock, ' ')
	if _, e := PlanLockedOwnerReassignment(u.Previous, u.Generated); e == nil {
		t.Fatal("mutated union anchor authorized")
	}
	if _, e := ReadLockedOwnerReassignmentState(filepath.Join(root, "absent"), []string{"left"}); e == nil {
		t.Fatal("missing root accepted")
	}
}
