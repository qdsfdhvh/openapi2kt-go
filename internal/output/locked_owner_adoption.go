package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// Old Kotlin capture has its own aggregate bound; it never increases the
// existing per-file budget or the core's document/graph/generation limits.
const maxAdoptionKotlinBytes = maxFileBytes

// LockedOwnerAdoptionState binds a read before input IO to one existing root,
// exact manifest, complete Kotlin inventory and raw owned lock. This observation
// is not a filesystem snapshot. Caller data must remain immutable in flight.
type LockedOwnerAdoptionState struct {
	Root, SourceRoot, Owner string
	Version                 uint32
	Manifest, NameLock      []byte
	Files                   []core.GeneratedFile
	OwnerNames              []string
	NameLocks               map[string][]byte
	Owners                  map[string][]string
	identity                os.FileInfo
}

type LockedOwnerAdoptionUpdate struct {
	Owner     string
	Previous  LockedOwnerAdoptionState
	Generated core.GenerationArtifacts
}

func lockedAdoptionInvalid() *diagnostic.Diagnostic {
	return fail("E_LOCKED_OWNER_ADOPTION_INVALID", "Locked ownership adoption requires one existing complete version3 root or an exact locked version4 owner inventory, with explicit single-spec or full-configuration selection.")
}

// ReadLockedOwnerAdoptionState is read-only, validates all source bytes and the
// dedicated destination, and captures canonical binding before source input IO.
func ReadLockedOwnerAdoptionState(root, owner string) (LockedOwnerAdoptionState, *diagnostic.Diagnostic) {
	return readLockedAdoptionState(root, []string{owner}, false)
}

// ReadSplitLockedAdoptionState captures complete v3 or exact multiowner v4
// state with the same aggregate/remaining-read Kotlin budget as single adoption.
func ReadSplitLockedAdoptionState(root string, owners []string) (LockedOwnerAdoptionState, *diagnostic.Diagnostic) {
	return readLockedAdoptionState(root, owners, true)
}

func readLockedAdoptionState(root string, owners []string, split bool) (LockedOwnerAdoptionState, *diagnostic.Diagnostic) {
	return readLockedAdoptionInventory(root, owners, split, "")
}

func readLockedAdoptionInventory(root string, owners []string, split bool, addition string) (LockedOwnerAdoptionState, *diagnostic.Diagnostic) {
	empty := LockedOwnerAdoptionState{}
	if len(owners) == 0 || len(owners) > maxLockedOwners {
		return empty, lockedAdoptionInvalid()
	}
	owners = append([]string(nil), owners...)
	sort.Strings(owners)
	for i, owner := range owners {
		if !ownerName(owner) || i > 0 && owners[i-1] == owner {
			return empty, lockedAdoptionInvalid()
		}
	}
	owner := owners[0]

	canonical, e := canonicalRoot(root)
	if e != nil {
		return empty, e
	}
	if e := inspectRecovery(canonical); e != nil {
		return empty, e
	}
	identity, err := os.Stat(canonical)
	if os.IsNotExist(err) {
		return empty, lockedAdoptionInvalid()
	}
	if err != nil {
		return empty, ioFailure()
	}
	raw, exists, e := readRegular(filepath.Join(canonical, manifestName))
	if e != nil {
		return empty, e
	}
	if !exists {
		return empty, lockedAdoptionInvalid()
	}
	m, e := parseLockedAdoptionManifest(raw)
	if e != nil {
		return empty, e
	}
	if addition != "" {
		if m.Version != 4 || m.Owners == nil || m.NameLocks == nil || (*m.Owners)[owner] == nil {
			return empty, lockedAdoptionInvalid()
		}
		owners = sortedKeys(*m.Owners)
		if _, exists := (*m.Owners)[addition]; !exists {
			if len(owners) >= maxLockedOwners {
				return empty, lockedAdoptionInvalid()
			}
			if _, err := readNamedState(canonical, addition, nil); err != nil {
				return empty, err
			}
		}
	}
	if m.Version != 3 && m.Version != 4 || len(m.Files) == 0 {
		return empty, lockedAdoptionInvalid()
	}
	source, err := filepath.Abs(root)
	if err != nil {
		return empty, ioFailure()
	}
	s := LockedOwnerAdoptionState{Owner: owner, Root: canonical, SourceRoot: source, Version: m.Version, Manifest: raw, Files: []core.GeneratedFile{}, identity: identity, OwnerNames: owners, NameLocks: map[string][]byte{}}
	if m.Version == 3 {
		s.NameLock, e = readManagedNameLock(canonical, m)
		if e == nil {
			for _, name := range owners {
				if _, e = readNamedState(canonical, name, nil); e != nil {
					break
				}
				s.NameLocks[name] = s.NameLock
			}
		}
	} else {
		if m.Owners == nil || m.NameLocks == nil || len(*m.Owners) != len(owners) || len(*m.NameLocks) != len(owners) {
			return empty, lockedAdoptionInvalid()
		}
		s.Owners = *m.Owners
		total := 0
		for _, name := range owners {
			if s.Owners[name] == nil || !split && len(s.Owners[name]) != len(m.Files) {
				return empty, lockedAdoptionInvalid()
			}
			hash, ok := (*m.NameLocks)[name]
			if !ok {
				return empty, lockedAdoptionInvalid()
			}
			payload, err := readNamedState(canonical, name, &hash)
			if err != nil {
				return empty, err
			}
			total += len(payload)
			if total > maxOwnerNameLockBytes {
				return empty, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
			}
			s.NameLocks[name] = payload
		}
		s.NameLock = s.NameLocks[owner]
		_, e = readManagedNameLock(canonical, nil)
	}
	if e != nil {
		return empty, e
	}

	paths := sortedKeys(m.Files)
	if e := inspectAdoptionKotlinBudget(canonical, paths); e != nil {
		return empty, e
	}
	// Capture gets fresh parent observations and still enforces the remaining
	// budget at each actual read, guarding growth since the size-only preflight.
	checked := map[string]bool{}
	total := 0
	for _, path := range paths {
		if e := inspectParentsWithCache(canonical, path, checked); e != nil {
			return empty, e
		}
		data, exists, e := readRegularLimited(filepath.Join(canonical, filepath.FromSlash(path)), int64(maxAdoptionKotlinBytes-total))
		if e != nil {
			return empty, e
		}
		if !exists {
			return empty, fail("E_OUTPUT_MISSING", "Restore missing manifest-owned Kotlin before locked ownership adoption.")
		}
		if digest(data) != m.Files[path] {
			return empty, fail("E_OUTPUT_MODIFIED", "Restore modified manifest-owned Kotlin before locked ownership adoption.")
		}
		total += len(data)
		s.Files = append(s.Files, core.GeneratedFile{Path: path, Content: string(data)})
	}
	return s, nil
}

// Check the complete inventory before allocating any Kotlin payload capture.
// Manifest/lock reads retain their own existing limits. This is a size budget,
// not an RSS bound or a snapshot against noncooperating filesystem changes.
func inspectAdoptionKotlinBudget(root string, paths []string) *diagnostic.Diagnostic {
	checked := map[string]bool{}
	var total int64
	for _, path := range paths {
		if e := inspectParentsWithCache(root, path, checked); e != nil {
			return e
		}
		info, e := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
		if os.IsNotExist(e) {
			return fail("E_OUTPUT_MISSING", "Restore missing manifest-owned Kotlin before locked ownership adoption.")
		}
		if e != nil {
			return ioFailure()
		}
		if !info.Mode().IsRegular() {
			return fail("E_OUTPUT_PATH", "Output entries must be regular files, not directories or symbolic links.")
		}
		if info.Size() > int64(maxFileBytes) || info.Size() > int64(maxAdoptionKotlinBytes)-total {
			return fail("E_OUTPUT_LIMIT", "Manifest-owned Kotlin capture exceeds the 64 MiB aggregate limit.")
		}
		total += info.Size()
	}
	return nil
}

func parseLockedAdoptionManifest(raw []byte) (*manifest, *diagnostic.Diagnostic) {
	m, e := parseManifestKind(raw, false, true)
	if e != nil && e.Code == "E_MANIFEST_VERSION" {
		return parseManifestKind(raw, true, true)
	}
	return m, e
}

// LockedOwnerAdoptionManifest returns the canonical candidate manifest bytes used
// by the existing transaction. Reporting and commit use exactly this encoding.
func LockedOwnerAdoptionManifest(s LockedOwnerAdoptionState, owner string) ([]byte, *diagnostic.Diagnostic) {
	if owner != s.Owner || !ownerName(owner) || s.Version != 3 && s.Version != 4 {
		return nil, lockedAdoptionInvalid()
	}
	m, e := parseLockedAdoptionManifest(s.Manifest)
	if e != nil {
		return nil, e
	}
	if m.Version == 4 {
		return s.Manifest, nil
	}
	paths := sortedKeys(m.Files)
	owners := map[string][]string{owner: paths}
	locks := map[string]string{owner: digest(s.NameLock)}
	next := manifest{Version: 4, Files: m.Files, Owners: &owners, NameLocks: &locks}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return nil, ioFailure()
	}
	return data, nil
}

func WriteLockedOwnerAdoption(root string, u LockedOwnerAdoptionUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(map[string]LockedOwnerAdoptionUpdate{root: u}, check, nil, makeLockedOwnerAdoptionPlan)
}

func makeLockedOwnerAdoptionPlan(root string, u LockedOwnerAdoptionUpdate) (*plan, *diagnostic.Diagnostic) {
	stale := func() (*plan, *diagnostic.Diagnostic) {
		return nil, fail("E_LOCKED_OWNER_ADOPTION_STALE", "Output root, ownership, Kotlin or raw name-lock bytes changed after reading; review current output before adoption again.")
	}
	if !ownerName(u.Owner) || u.Owner != u.Previous.Owner || u.Previous.Root == "" || u.Previous.SourceRoot == "" || u.Previous.identity == nil {
		return nil, lockedAdoptionInvalid()
	}
	bound, e := canonicalRoot(u.Previous.SourceRoot)
	if e != nil {
		return nil, e
	}
	if bound != u.Previous.Root || root != bound {
		return stale()
	}
	s, e := ReadLockedOwnerAdoptionState(u.Previous.SourceRoot, u.Owner)
	if e != nil {
		return nil, e
	}
	if s.Version != u.Previous.Version || !os.SameFile(s.identity, u.Previous.identity) || !bytes.Equal(s.Manifest, u.Previous.Manifest) || !bytes.Equal(s.NameLock, u.Previous.NameLock) || !reflect.DeepEqual(s.Files, u.Previous.Files) {
		return stale()
	}
	candidate := append([]core.GeneratedFile(nil), u.Generated.Files...)
	sort.Slice(candidate, func(i, j int) bool { return candidate[i].Path < candidate[j].Path })
	if !reflect.DeepEqual(candidate, s.Files) || u.Generated.NameLock == nil {
		return nil, fail("E_LOCKED_OWNER_ADOPTION_CHANGED", "Locked ownership adoption must preserve the entire Kotlin inventory and existing lock mode; update the spec separately.")
	}
	before, e := parseNameLockBytes(s.NameLock)
	if e != nil {
		return nil, e
	}
	after, e := parseNameLockBytes(u.Generated.NameLock)
	if e != nil {
		return nil, e
	}
	if before.Profile != after.Profile {
		return nil, fail("E_NAME_LOCK_PROFILE", "Locked ownership adoption requires the stored package, suffix and qualification profile.")
	}
	if !reflect.DeepEqual(before, after) {
		return nil, fail("E_LOCKED_OWNER_ADOPTION_CHANGED", "Locked ownership adoption must preserve allocations and original occurrences; update the spec separately.")
	}
	raw, e := LockedOwnerAdoptionManifest(s, u.Owner)
	if e != nil {
		return nil, e
	}
	m, e := parseManifestKind(raw, true, true)
	if e != nil {
		return nil, e
	}
	p := &plan{Writes: map[string][]byte{}, Manifest: *m, Changed: s.Version == 3}
	if p.Changed {
		p.Writes[ownerNameLockFile(u.Owner)] = s.NameLock
		p.Removals = []string{nameLockFile}
	}
	return p, nil
}
