package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"openapi-kt/internal/schema"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNameLockOwnersExpectedInventoryAndDestinations(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, kind := range []string{"missing-key", "extra-key", "expected-empty", "unowned", "symlink", "missing", "edited", "malformed", "unsupported", "disabled-empty"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": a}, map[string][]byte{"alpha": nil}, false)
			code := "E_OWNER_UPDATE_INVALID"
			switch kind {
			case "missing-key":
				u.PreviousNameLocks = nil
			case "extra-key":
				u.PreviousNameLocks["beta"] = nil
			case "expected-empty":
				u.PreviousNameLocks["alpha"] = []byte{}
				code = "E_NAME_LOCK_STALE"
			case "unowned":
				put(t, filepath.Join(root, ownerNameLockFile("alpha")), a.NameLock)
				code = "E_OUTPUT_COLLISION"
			case "symlink":
				put(t, filepath.Join(root, "sentinel"), a.NameLock)
				if e := os.Symlink("sentinel", filepath.Join(root, ownerNameLockFile("alpha"))); e != nil {
					t.Skip("symlinks unavailable")
				}
				code = "E_OUTPUT_PATH"
			default:
				if _, e := WriteNameLockedOwners(root, u, false); e != nil {
					t.Fatal(e)
				}
				u.PreviousNameLocks["alpha"] = a.NameLock
				switch kind {
				case "missing":
					_ = os.Remove(filepath.Join(root, ownerNameLockFile("alpha")))
					code = "E_NAME_LOCK_MISSING"
				case "edited":
					put(t, filepath.Join(root, ownerNameLockFile("alpha")), []byte("synthetic manually edited"))
					code = "E_NAME_LOCK_MODIFIED"
				case "disabled-empty":
					u.Generated["alpha"] = core.GenerationArtifacts{}
					u.PreviousNameLocks = nil
					code = "E_NAME_LOCK_DISABLED"
				case "malformed", "unsupported":
					state := []byte(`{}`)
					code = "E_NAME_LOCK_INVALID"
					if kind == "unsupported" {
						state = []byte(strings.Replace(string(a.NameLock), `"version":1`, `"version":2`, 1))
						code = "E_NAME_LOCK_VERSION"
					}
					m, e := readManifestKind(root, true, true)
					if e != nil {
						t.Fatal(e)
					}
					(*m.NameLocks)["alpha"] = digest(state)
					b, _ := json.Marshal(m)
					put(t, filepath.Join(root, manifestName), b)
					put(t, filepath.Join(root, ownerNameLockFile("alpha")), state)
				}
			}
			before := snapshot(t, root)
			for _, check := range []bool{true, false} {
				_, e := WriteNameLockedOwners(root, u, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("rejection changed old root")
				}
			}
		})
	}
	root := filepath.Join(t.TempDir(), "absent")
	bad := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": a}, nil, false)
	_, e := WriteNameLockedOwners(root, bad, false)
	errorCode(t, e, "E_OWNER_UPDATE_INVALID")
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("invalid expected inventory created root")
	}
	for _, owner := range []string{"../alpha", "Con", "Alpha", "alpha"} {
		if owner == "../alpha" {
			_, e := ReadOwnerNameLock(root, owner)
			errorCode(t, e, "E_OWNER_UPDATE_INVALID")
			continue
		}
		f := ownerNameLockFile(owner)
		if len(f) != len(ownerNameLockFile("alpha")) || strings.ContainsAny(f, "/\\:") {
			t.Fatal("unsafe derived metadata path")
		}
	}
	if ownerNameLockFile("Alpha") == ownerNameLockFile("alpha") {
		t.Fatal("lost exact-case owner identity")
	}
}
func TestNameLockOwnersStrictVersionFour(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	hash := digest(a.NameLock)
	files := map[string]string{}
	paths := []string{}
	for _, f := range a.Files {
		files[f.Path] = digest([]byte(f.Content))
		paths = append(paths, f.Path)
	}
	filesJSON, _ := json.Marshal(files)
	ownersJSON, _ := json.Marshal(map[string][]string{"alpha": paths})
	// Incoming core artifact order is sorted; owner membership independently is
	// validated by the existing strict sorted-path parser.
	for _, c := range []struct{ locks, extra, code string }{
		{`null`, "", "E_MANIFEST_INVALID"}, {`{"alpha":null}`, "", "E_MANIFEST_INVALID"}, {`{"alpha":"bad"}`, "", "E_MANIFEST_INVALID"}, {`{"beta":"` + hash + `"}`, "", "E_MANIFEST_INVALID"}, {`{"alpha":"` + hash + `","alpha":"` + hash + `"}`, "", "E_MANIFEST_INVALID"}, {`{"../alpha":"` + hash + `"}`, "", "E_MANIFEST_INVALID"}, {`{"alpha":"` + hash + `"}`, `,"extra":true`, "E_MANIFEST_INVALID"},
	} {
		root := t.TempDir()
		data := `{"version":4,"files":` + string(filesJSON) + `,"owners":` + string(ownersJSON) + `,"name_locks":` + c.locks + c.extra + `}`
		put(t, filepath.Join(root, manifestName), []byte(data))
		before := snapshot(t, root)
		_, e := ReadOwnerNameLock(root, "alpha")
		errorCode(t, e, c.code)
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("reader wrote")
		}
	}
	root := t.TempDir()
	if _, e := WriteNameLocked(root, a, nil, false); e != nil {
		t.Fatal(e)
	}
	_, e := ReadOwnerNameLock(root, "alpha")
	errorCode(t, e, "E_MANIFEST_VERSION")
	u := lockedUpdate([]string{"alpha"}, map[string]core.GenerationArtifacts{"alpha": a}, map[string][]byte{"alpha": nil}, false)
	u.AdoptLegacy = "alpha"
	_, e = WriteNameLockedOwners(root, u, false)
	errorCode(t, e, "E_MANIFEST_VERSION")
}
func TestNameLockOwnersStorageBudgetBoundaries(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	u := lockedUpdate(nil, map[string]core.GenerationArtifacts{}, map[string][]byte{}, false)
	for i := 0; i < maxLockedOwners; i++ {
		owner := fmt.Sprintf("owner%03d", i)
		u.ActiveOwners = append(u.ActiveOwners, owner)
		u.Generated[owner] = a
		u.PreviousNameLocks[owner] = nil
	}
	root := filepath.Join(t.TempDir(), "absent")
	if changed, e := WriteNameLockedOwners(root, u, true); e != nil || !changed {
		t.Fatal("256 owner boundary", e)
	}
	u.ActiveOwners = append(u.ActiveOwners, "overflow")
	u.Generated["overflow"] = a
	u.PreviousNameLocks["overflow"] = nil
	_, e := WriteNameLockedOwners(root, u, false)
	errorCode(t, e, "E_NAME_LOCK_LIMIT")
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("owner cap created IO")
	}
	padded := a
	padded.NameLock = append(append([]byte(nil), a.NameLock...), bytes.Repeat([]byte(" "), schema.MaxNameLockBytes-len(a.NameLock))...)
	u = lockedUpdate(nil, map[string]core.GenerationArtifacts{}, map[string][]byte{}, false)
	for i := 0; i < 16; i++ {
		owner := fmt.Sprintf("large%02d", i)
		u.ActiveOwners = append(u.ActiveOwners, owner)
		u.Generated[owner] = padded
		u.PreviousNameLocks[owner] = nil
	}
	root = t.TempDir()
	if _, e := WriteNameLockedOwners(root, u, false); e != nil {
		t.Fatal("16MiB storage boundary", e)
	}
	state, e := ReadOwnerNameLock(root, "large00")
	if e != nil || len(state) != schema.MaxNameLockBytes {
		t.Fatal("1MiB read boundary", e)
	}
	before := snapshot(t, root)
	small := a
	small.NameLock = append(append([]byte(nil), a.NameLock...), ' ')
	u.ActiveOwners = append(u.ActiveOwners, "overflow")
	u.Generated["overflow"] = small
	trimmed := padded
	trimmed.NameLock = padded.NameLock[:schema.MaxNameLockBytes-len(small.NameLock)+1]
	u.Generated["large15"] = trimmed // exactly 16 MiB + 1 across the 17 states
	u.PreviousNameLocks["overflow"] = nil
	for owner := range u.Generated {
		if owner != "overflow" {
			u.PreviousNameLocks[owner] = padded.NameLock
		}
	}
	_, e = WriteNameLockedOwners(root, u, false)
	errorCode(t, e, "E_NAME_LOCK_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("aggregate overflow changed old state")
	}
	// Inject an internally hashed but oversized OLD aggregate. Reconciliation
	// cannot delete it to get below the cap without first validating old state.
	m, e := readManifestKind(root, true, true)
	if e != nil {
		t.Fatal(e)
	}
	(*m.Owners)["overflow"] = append([]string(nil), (*m.Owners)["large00"]...)
	(*m.NameLocks)["overflow"] = digest(small.NameLock)
	(*m.NameLocks)["large15"] = digest(trimmed.NameLock)
	put(t, filepath.Join(root, ownerNameLockFile("large15")), trimmed.NameLock)
	data, _ := json.Marshal(m)
	put(t, filepath.Join(root, manifestName), data)
	put(t, filepath.Join(root, ownerNameLockFile("overflow")), small.NameLock)
	before = snapshot(t, root)
	_, e = WriteNameLockedOwners(root, lockedUpdate(nil, nil, nil, true), false)
	errorCode(t, e, "E_NAME_LOCK_LIMIT")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("oversized old inventory deleted")
	}
}
func FuzzNameLockOwnersManifest(f *testing.F) {
	hash := strings.Repeat("0", 64)
	for _, s := range []string{`{"version":4,"files":{},"owners":{},"name_locks":{}}`, `{"version":4,"files":{"A.kt":"` + hash + `"},"owners":{"alpha":["A.kt"]},"name_locks":{"alpha":"` + hash + `"}}`, `{"version":4,"files":{},"owners":{},"name_locks":null}`, `{}`} {
		f.Add(s)
	}
	root := f.TempDir()
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 8192 {
			t.Skip()
		}
		put(t, filepath.Join(root, manifestName), []byte(s))
		m, e := readManifestKind(root, true, true)
		if e == nil {
			if m == nil || m.Owners == nil || m.Version != 2 && m.Version != 4 {
				t.Fatal("invalid accepted owner manifest")
			}
			if m.Version == 4 {
				if m.NameLocks == nil || len(*m.NameLocks) > maxLockedOwners {
					t.Fatal("invalid accepted name-lock inventory")
				}
				for owner, hash := range *m.NameLocks {
					if !ownerName(owner) || len((*m.Owners)[owner]) == 0 || !validContentHash(hash) {
						t.Fatal("dangling/unsafe state claim")
					}
				}
			}
		}
	})
}
