package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/schema"
)

func TestRetainedLockedAdoptionResultOwnerCountBoundary(t *testing.T) {
	root, u := retainedAdoptionFixture(t, []string{"part"})
	names := []string{}
	for i := 0; i < maxLockedOwners-1; i++ {
		names = append(names, fmt.Sprintf("candidate%03d", i))
	}
	s, e := ReadRetainedLockedAdoptionState(root, "original", names)
	if e != nil {
		t.Fatal("exact owner boundary refused", e)
	}
	parsed, e := parseNameLockBytes(s.NameLock)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := (&schema.HoistedNameLock{Version: 1, Profile: parsed.Profile, Entries: []schema.NameLockEntry{}}).Encode()
	if e != nil {
		t.Fatal(e)
	}
	generated := map[string]core.GenerationArtifacts{}
	for _, name := range names {
		g := u.Generated["part"]
		g.NameLock = raw
		generated[name] = g
	}
	before := snapshot(t, root)
	next, e := PlanRetainedLockedAdoption(s, generated)
	if e != nil || len(next.NameLocks) != maxLockedOwners {
		t.Fatal("exact256 result owners refused", e)
	}
	if _, e := ReadRetainedLockedAdoptionState(root, "original", append(names, "oneextra")); e == nil {
		t.Fatal("owner boundary exceeded")
	}
	for _, names := range [][]string{nil, {"original"}, {"part", "part"}, {"../invalid"}} {
		if _, e := ReadRetainedLockedAdoptionState(root, "original", names); e == nil {
			t.Fatal("invalid original/candidate identity accepted", names)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("owner preflight wrote")
	}
}

func TestRetainedLockedAdoptionActualRawDonorBudget(t *testing.T) {
	for _, over := range []bool{false, true} {
		root, _ := retainedAdoptionFixture(t, []string{"part"})
		raw := mustRead(t, filepath.Join(root, nameLockFile))
		raw = append(raw, bytes.Repeat([]byte(" "), schema.MaxNameLockBytes-len(raw))...)
		put(t, filepath.Join(root, nameLockFile), raw)
		m, e := readManifestKind(root, false, true)
		if e != nil {
			t.Fatal(e)
		}
		hash := digest(raw)
		m.NameLock = &hash
		manifest, _ := json.MarshalIndent(m, "", "  ")
		put(t, filepath.Join(root, manifestName), manifest)
		count := 15
		if over {
			count++
		}
		names := []string{}
		for i := 0; i < count; i++ {
			names = append(names, fmt.Sprintf("owner%02d", i))
		}
		s, e := ReadRetainedLockedAdoptionState(root, "original", names)
		if e != nil {
			t.Fatal(e)
		}
		state, e := parseNameLockBytes(s.NameLock)
		if e != nil {
			t.Fatal(e)
		}
		canonical, e := state.Encode()
		if e != nil {
			t.Fatal(e)
		}
		generated := map[string]core.GenerationArtifacts{}
		for _, name := range names {
			generated[name] = core.GenerationArtifacts{Files: s.Previous.Files, NameLock: canonical}
		}
		before := snapshot(t, root)
		next, e := PlanRetainedLockedAdoption(s, generated)
		if over {
			errorCode(t, e, "E_NAME_LOCK_LIMIT")
		} else {
			if e != nil {
				t.Fatal("exact16MiB actual donor budget refused", e)
			}
			sum := 0
			for _, payload := range next.NameLocks {
				sum += len(payload)
			}
			if sum != maxOwnerNameLockBytes {
				t.Fatal("test did not reach actual raw budget", sum)
			}
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("raw donor preflight wrote")
		}
	}
}

func TestRetainedLockedAdoptionV4RetainerMustOwnCompleteOriginal(t *testing.T) {
	root, u := retainedAdoptionFixture(t, []string{"part"})
	if _, e := WriteRetainedLockedAdoption(root, u, false); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	if _, e := ReadRetainedLockedAdoptionState(root, "part", []string{"new"}); e == nil {
		t.Fatal("partial peer became fake complete source authority")
	}
	s, e := ReadRetainedLockedAdoptionState(root, "original", []string{"part"})
	if e != nil {
		t.Fatal(e)
	}
	g := u.Generated["part"]
	g.Files = s.Previous.Files
	if _, e := PlanRetainedLockedAdoption(s, map[string]core.GenerationArtifacts{"part": g}); e == nil {
		t.Fatal("existing peer membership silently reassigned")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("invalid v4 retainer/replay wrote")
	}
}
