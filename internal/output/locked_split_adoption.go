package output

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

type SplitLockedAdoptionUpdate struct {
	Previous  LockedOwnerAdoptionState
	Generated map[string]core.GenerationArtifacts
}

// SplitLockedAdoptionMetadata is a detached, deterministic metadata candidate.
// Kotlin is verified byte-for-byte and is never written by this action.
type SplitLockedAdoptionMetadata struct {
	Manifest  []byte
	NameLocks map[string][]byte
}

func splitAdoptionChanged() *diagnostic.Diagnostic {
	return fail("E_LOCKED_OWNER_ADOPTION_CHANGED", "Split adoption must preserve complete original Kotlin bytes, allocation names and the union of original occurrences.")
}

// PlanSplitLockedAdoption verifies genuine partitions and shared memberships.
// Both original capture and aggregate candidate Kotlin remain bounded at 64 MiB;
// persisted owner/lock/occurrence limits retain their existing values.
func PlanSplitLockedAdoption(s LockedOwnerAdoptionState, generated map[string]core.GenerationArtifacts) (SplitLockedAdoptionMetadata, *diagnostic.Diagnostic) {
	empty := SplitLockedAdoptionMetadata{}
	if len(s.OwnerNames) == 0 || len(s.OwnerNames) > maxLockedOwners || len(generated) != len(s.OwnerNames) {
		return empty, lockedAdoptionInvalid()
	}
	m, err := parseLockedAdoptionManifest(s.Manifest)
	if err != nil {
		return empty, err
	}
	if m.Version != s.Version || s.Version != 3 && s.Version != 4 {
		return empty, lockedAdoptionInvalid()
	}
	original := map[string]core.GeneratedFile{}
	for _, file := range s.Files {
		if original[file.Path].Path != "" || m.Files[file.Path] != digest([]byte(file.Content)) {
			return empty, splitAdoptionChanged()
		}
		original[file.Path] = file
	}
	if len(original) != len(m.Files) || len(original) == 0 {
		return empty, splitAdoptionChanged()
	}
	if s.Version == 4 && (m.Owners == nil || !reflect.DeepEqual(s.Owners, *m.Owners)) {
		return empty, lockedAdoptionInvalid()
	}
	anchor, err := parseNameLockBytes(s.NameLock)
	if err != nil {
		return empty, err
	}
	type allocation struct {
		name    string
		origins map[string]bool
	}
	allocations := map[string]allocation{}
	for _, entry := range anchor.Entries {
		origins := map[string]bool{}
		for _, origin := range entry.Occurrences {
			origins[origin] = true
		}
		allocations[entry.Role+":"+entry.Shape] = allocation{entry.Name, origins}
	}
	coverage := map[string]map[string]bool{}
	owners, hashes := map[string][]string{}, map[string]string{}
	payloads := map[string][]byte{}
	files := map[string]string{}
	lockBytes := 0
	budget, err := NewSplitAdoptionBudget(s.Files)
	if err != nil {
		return empty, err
	}
	for i, owner := range s.OwnerNames {
		if !ownerName(owner) || i > 0 && owner <= s.OwnerNames[i-1] {
			return empty, lockedAdoptionInvalid()
		}
		candidate, ok := generated[owner]
		if !ok || len(candidate.Files) == 0 || candidate.NameLock == nil {
			return empty, lockedAdoptionInvalid()
		}
		if err := budget.Add(owner, candidate); err != nil {
			return empty, err
		}
		prior := s.NameLocks[owner]
		if s.Version == 3 {
			if m.NameLock == nil || *m.NameLock != digest(s.NameLock) || !bytes.Equal(prior, s.NameLock) {
				return empty, lockedAdoptionInvalid()
			}
		} else {
			if m.NameLocks == nil || m.Owners == nil || (*m.NameLocks)[owner] != digest(prior) {
				return empty, lockedAdoptionInvalid()
			}
		}
		before, err := parseNameLockBytes(prior)
		if err != nil {
			return empty, err
		}
		after, err := parseNameLockBytes(candidate.NameLock)
		if err != nil {
			return empty, err
		}
		if before.Profile != after.Profile || before.Profile != anchor.Profile {
			return empty, fail("E_NAME_LOCK_PROFILE", "Split adoption requires each original name-lock profile.")
		}
		if s.Version == 4 && !reflect.DeepEqual(before, after) {
			return empty, splitAdoptionChanged()
		}
		for _, entry := range after.Entries {
			key := entry.Role + ":" + entry.Shape
			if s.Version == 3 {
				old, ok := allocations[key]
				if !ok || old.name != entry.Name {
					return empty, splitAdoptionChanged()
				}
				for _, origin := range entry.Occurrences {
					if !old.origins[origin] {
						return empty, splitAdoptionChanged()
					}
				}
			}
			if coverage[key] == nil {
				coverage[key] = map[string]bool{}
			}
			for _, origin := range entry.Occurrences {
				coverage[key][origin] = true
			}
		}
		paths := []string{}
		seen := map[string]bool{}
		for _, file := range candidate.Files {
			if relativePath(file.Path) != nil || seen[file.Path] || original[file.Path] != file {
				return empty, splitAdoptionChanged()
			}
			seen[file.Path] = true
			paths = append(paths, file.Path)
			files[file.Path] = m.Files[file.Path]
		}
		sort.Strings(paths)
		if s.Version == 4 && !reflect.DeepEqual(paths, s.Owners[owner]) {
			return empty, splitAdoptionChanged()
		}
		owners[owner] = paths
		payload := candidate.NameLock
		if reflect.DeepEqual(before, after) {
			payload = prior
		}
		if len(payload) > maxOwnerNameLockBytes-lockBytes {
			return empty, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
		lockBytes += len(payload)
		payloads[owner] = payload
		hashes[owner] = digest(payload)
	}
	if len(files) != len(m.Files) {
		return empty, splitAdoptionChanged()
	}
	if s.Version == 3 {
		if len(coverage) != len(allocations) {
			return empty, splitAdoptionChanged()
		}
		for key, entry := range allocations {
			if len(coverage[key]) != len(entry.origins) {
				return empty, splitAdoptionChanged()
			}
			for origin := range entry.origins {
				if !coverage[key][origin] {
					return empty, splitAdoptionChanged()
				}
			}
		}
	} else {
		if len(s.Owners) != len(owners) || len(s.NameLocks) != len(payloads) {
			return empty, lockedAdoptionInvalid()
		}
		return SplitLockedAdoptionMetadata{Manifest: s.Manifest, NameLocks: payloads}, nil
	}
	next := manifest{Version: 4, Files: files, Owners: &owners, NameLocks: &hashes}
	raw, e := json.MarshalIndent(next, "", "  ")
	if e != nil {
		return empty, ioFailure()
	}
	if _, err := parseManifestKind(raw, true, true); err != nil {
		return empty, err
	}
	return SplitLockedAdoptionMetadata{Manifest: raw, NameLocks: payloads}, nil
}

func WriteSplitLockedAdoption(root string, u SplitLockedAdoptionUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(map[string]SplitLockedAdoptionUpdate{root: u}, check, nil, makeSplitLockedAdoptionPlan)
}

func makeSplitLockedAdoptionPlan(root string, u SplitLockedAdoptionUpdate) (*plan, *diagnostic.Diagnostic) {
	prior := u.Previous
	if prior.identity == nil || prior.Root == "" || prior.SourceRoot == "" {
		return nil, lockedAdoptionInvalid()
	}
	bound, err := canonicalRoot(prior.SourceRoot)
	if err != nil {
		return nil, err
	}
	stale := func() *diagnostic.Diagnostic {
		return fail("E_LOCKED_OWNER_ADOPTION_STALE", "Original root, ownership, Kotlin or raw locks changed after split-adoption planning.")
	}
	if bound != prior.Root || root != bound {
		return nil, stale()
	}
	current, err := ReadSplitLockedAdoptionState(prior.SourceRoot, prior.OwnerNames)
	if err != nil {
		return nil, err
	}
	if current.Version != prior.Version || !os.SameFile(current.identity, prior.identity) || !bytes.Equal(current.Manifest, prior.Manifest) || !reflect.DeepEqual(current.NameLocks, prior.NameLocks) || !reflect.DeepEqual(current.Files, prior.Files) || !reflect.DeepEqual(current.Owners, prior.Owners) {
		return nil, stale()
	}
	next, err := PlanSplitLockedAdoption(current, u.Generated)
	if err != nil {
		return nil, err
	}
	p := &plan{Writes: map[string][]byte{}, ManifestBytes: next.Manifest, Changed: current.Version == 3}
	if p.Changed {
		for _, owner := range current.OwnerNames {
			if _, err := readNamedState(root, owner, nil); err != nil {
				return nil, err
			}
			p.Writes[ownerNameLockFile(owner)] = next.NameLocks[owner]
		}
		p.Removals = []string{nameLockFile}
	}
	return p, nil
}
