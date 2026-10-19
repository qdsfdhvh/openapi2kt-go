package output

import (
	"bytes"
	"path/filepath"
	"reflect"
	"sort"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// OwnerRenameState is detached, verified existing v2/v4 output. Manifest and
// every owned lock's exact bytes bind the eventual transaction to this read.
// Files are manifest-owned Kotlin only; no directory or writer lock is created.
// This is a point-in-time observation, not a filesystem snapshot.
type OwnerRenameState struct {
	Manifest  []byte
	Version   uint32
	Owners    map[string][]core.GeneratedFile
	NameLocks map[string][]byte
}

// OwnerRenameUpdate maps each new owner to its old owner. Every selected candidate
// must exactly preserve old Kotlin membership/bytes and lock semantics. Other
// owners remain retained, even when absent from the current config. No reconcile,
// root discovery, adoption, allocation reset or format transition is performed.
// Caller maps, artifacts and byte slices must remain immutable in flight.
type OwnerRenameUpdate struct {
	Previous  OwnerRenameState
	Renames   map[string]string
	Generated map[string]core.GenerationArtifacts
}

func ReadOwnerRenameState(root string) (OwnerRenameState, *diagnostic.Diagnostic) {
	s := OwnerRenameState{}
	if e := validateRoot(root); e != nil {
		return s, e
	}
	if e := inspectRecovery(root); e != nil {
		return s, e
	}
	data, exists, e := readRegular(filepath.Join(root, manifestName))
	if e != nil {
		return s, e
	}
	if !exists {
		return s, fail("E_OWNER_RENAME_INVALID", "Owner rename requires an existing version2 or version4 named-owner manifest.")
	}
	m, e := parseManifestKind(data, true, true)
	if e != nil {
		return s, e
	}
	s.Manifest = data
	s.Version = m.Version
	s.Owners = map[string][]core.GeneratedFile{}
	s.NameLocks = map[string][]byte{}
	// Read each file once, preserving all retained/shared membership. Missing and
	// modified bytes never become fresh generated replacements during migration.
	files := map[string]core.GeneratedFile{}
	checked := map[string]bool{}
	for _, path := range sortedKeys(m.Files) {
		if e := inspectParentsWithCache(root, path, checked); e != nil {
			return OwnerRenameState{}, e
		}
		data, exists, e := readRegular(filepath.Join(root, filepath.FromSlash(path)))
		if e != nil {
			return OwnerRenameState{}, e
		}
		if !exists {
			return OwnerRenameState{}, fail("E_OUTPUT_MISSING", "Restore missing manifest-owned Kotlin before owner rename.")
		}
		if digest(data) != m.Files[path] {
			return OwnerRenameState{}, fail("E_OUTPUT_MODIFIED", "Restore modified manifest-owned Kotlin before owner rename.")
		}
		files[path] = core.GeneratedFile{Path: path, Content: string(data)}
	}
	for _, owner := range sortedKeys(*m.Owners) {
		for _, path := range (*m.Owners)[owner] {
			s.Owners[owner] = append(s.Owners[owner], files[path])
		}
	}
	total := 0
	if m.NameLocks != nil {
		for _, owner := range sortedKeys(*m.NameLocks) {
			hash := (*m.NameLocks)[owner]
			data, e := readNamedState(root, owner, &hash)
			if e != nil {
				return OwnerRenameState{}, e
			}
			total += len(data)
			if total > maxOwnerNameLockBytes {
				return OwnerRenameState{}, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
			}
			s.NameLocks[owner] = data
		}
	}
	return s, nil
}

func WriteOwnerRenamesBatch(updates map[string]OwnerRenameUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(updates, check, nil, makeOwnerRenamePlan)
}

func makeOwnerRenamePlan(root string, u OwnerRenameUpdate) (*plan, *diagnostic.Diagnostic) {
	invalid := func() (*plan, *diagnostic.Diagnostic) {
		return nil, fail("E_OWNER_RENAME_INVALID", "Map every selected new owner to one distinct existing old owner; occupied targets must also be selected sources in the same root.")
	}
	if len(u.Renames) == 0 || len(u.Generated) != len(u.Renames) {
		return invalid()
	}
	sources := map[string]bool{}
	for _, next := range sortedKeys(u.Renames) {
		old := u.Renames[next]
		if !ownerName(next) || !ownerName(old) || sources[old] {
			return invalid()
		}
		if _, ok := u.Generated[next]; !ok {
			return invalid()
		}
		sources[old] = true
	}
	s, e := ReadOwnerRenameState(root)
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(s.Manifest, u.Previous.Manifest) || !reflect.DeepEqual(s.NameLocks, u.Previous.NameLocks) {
		return nil, fail("E_OWNER_RENAME_STALE", "Output ownership or name-lock state changed after planning; review the current state before renaming again.")
	}
	m, e := parseManifestKind(s.Manifest, true, true)
	if e != nil {
		return nil, e
	}
	p := &plan{Writes: map[string][]byte{}, Manifest: *m}
	for _, next := range sortedKeys(u.Renames) {
		old := u.Renames[next]
		oldFiles, exists := s.Owners[old]
		if !exists {
			return invalid()
		}
		if _, occupied := s.Owners[next]; occupied && old != next && !sources[next] {
			return invalid()
		}
		a := u.Generated[next]
		candidate := append([]core.GeneratedFile(nil), a.Files...)
		sort.Slice(candidate, func(i, j int) bool { return candidate[i].Path < candidate[j].Path })
		if !reflect.DeepEqual(oldFiles, candidate) {
			return nil, fail("E_OWNER_RENAME_CHANGED", "Owner rename must preserve exact Kotlin membership and bytes; update the spec separately.")
		}
		prior, locked := s.NameLocks[old]
		if locked != (a.NameLock != nil) {
			return nil, fail("E_OWNER_RENAME_CHANGED", "Owner rename must preserve the existing name-lock mode.")
		}
		if !locked {
			if _, e := readNamedState(root, old, nil); e != nil {
				return nil, e
			}
		}
		if locked {
			before, e := parseNameLockBytes(prior)
			if e != nil {
				return nil, e
			}
			after, e := parseNameLockBytes(a.NameLock)
			if e != nil {
				return nil, e
			}
			if before.Profile != after.Profile {
				return nil, fail("E_NAME_LOCK_PROFILE", "Owner rename requires the stored package, suffix and qualification profile.")
			}
			if !reflect.DeepEqual(before, after) {
				return nil, fail("E_OWNER_RENAME_CHANGED", "Owner rename must preserve name-lock allocations and occurrences; update the spec separately.")
			}
		}
		if old == next {
			continue
		}
		// An occupied destination is already captured and must be another
		// selected source. All other dedicated paths remain unowned collisions.
		if _, targetLocked := s.NameLocks[next]; !targetLocked {
			if _, e := readNamedState(root, next, nil); e != nil {
				return nil, e
			}
		}
		if locked {
			p.Writes[ownerNameLockFile(next)] = prior
			p.Removals = append(p.Removals, ownerNameLockFile(old))
		}
		p.Changed = true
	}
	if p.Changed {
		// Build from the original inventory in two passes. Incremental edits
		// could overwrite a source before a later swap/chain reads it.
		owners := map[string][]string{}
		locks := map[string]string{}
		for owner, paths := range *m.Owners {
			owners[owner] = paths
		}
		if m.NameLocks != nil {
			for owner, hash := range *m.NameLocks {
				locks[owner] = hash
			}
		}
		for old := range sources {
			delete(owners, old)
			delete(locks, old)
		}
		for next, old := range u.Renames {
			owners[next] = (*m.Owners)[old]
			if m.NameLocks != nil {
				if hash, locked := (*m.NameLocks)[old]; locked {
					locks[next] = hash
				}
			}
		}
		p.Manifest.Owners = &owners
		if m.NameLocks != nil {
			p.Manifest.NameLocks = &locks
		}
		kept := p.Removals[:0]
		for _, path := range p.Removals {
			if _, overwritten := p.Writes[path]; !overwritten {
				kept = append(kept, path)
			}
		}
		p.Removals = kept
		// A preserved target payload needs no write. Filter removals first:
		// an identical incoming payload must still keep that destination alive.
		for owner, prior := range s.NameLocks {
			path := ownerNameLockFile(owner)
			if next, written := p.Writes[path]; written && bytes.Equal(prior, next) {
				delete(p.Writes, path)
			}
		}
		p.Changed = !reflect.DeepEqual(p.Manifest, *m) || len(p.Writes) != 0 || len(p.Removals) != 0
	}
	sort.Strings(p.Removals)
	return p, nil
}
