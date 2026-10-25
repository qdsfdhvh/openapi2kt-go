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

// ExistingLockedAdoptionState captures the complete old v4 inventory before
// input IO. Donor is explicit; peers need not appear in the new configuration.
// Captures and generated artifacts must remain immutable during planning.
type ExistingLockedAdoptionState struct {
	Previous     LockedOwnerAdoptionState
	Owner, Donor string
}

type ExistingLockedAdoptionUpdate struct {
	Previous  ExistingLockedAdoptionState
	Generated core.GenerationArtifacts
}

func ReadExistingLockedAdoptionState(root, owner, donor string) (ExistingLockedAdoptionState, *diagnostic.Diagnostic) {
	if !ownerName(owner) || !ownerName(donor) || owner == donor {
		return ExistingLockedAdoptionState{}, lockedAdoptionInvalid()
	}
	prior, err := readLockedAdoptionInventory(root, []string{donor}, true, owner)
	if err != nil {
		return ExistingLockedAdoptionState{}, err
	}
	return ExistingLockedAdoptionState{Previous: prior, Owner: owner, Donor: donor}, nil
}

// PlanExistingLockedAdoption adds one shared/subset membership. All old owner
// memberships and raw lock payloads stay unchanged; no Kotlin is written.
func PlanExistingLockedAdoption(s ExistingLockedAdoptionState, generated core.GenerationArtifacts) (SplitLockedAdoptionMetadata, *diagnostic.Diagnostic) {
	empty := SplitLockedAdoptionMetadata{}
	p := s.Previous
	if !ownerName(s.Owner) || !ownerName(s.Donor) || s.Owner == s.Donor || p.Owner != s.Donor || p.Version != 4 {
		return empty, lockedAdoptionInvalid()
	}
	m, err := parseManifestKind(p.Manifest, true, true)
	if err != nil {
		return empty, err
	}
	if m.Version != 4 || m.Owners == nil || m.NameLocks == nil || !reflect.DeepEqual(p.Owners, *m.Owners) || !reflect.DeepEqual(p.OwnerNames, sortedKeys(p.Owners)) || len(p.NameLocks) != len(p.Owners) || len(p.OwnerNames) > maxLockedOwners || p.Owners[s.Donor] == nil {
		return empty, lockedAdoptionInvalid()
	}
	lockBytes := 0
	for _, name := range p.OwnerNames {
		payload := p.NameLocks[name]
		if (*m.NameLocks)[name] != digest(payload) || len(payload) > maxOwnerNameLockBytes-lockBytes {
			return empty, lockedAdoptionInvalid()
		}
		lockBytes += len(payload)
	}
	budget, err := NewSplitAdoptionBudget(p.Files)
	if err != nil {
		return empty, err
	}
	// Count all retained peer memberships and locks before allocating the new
	// manifest, even though peers contribute no candidate Kotlin payloads.
	for _, owner := range p.OwnerNames {
		files := []core.GeneratedFile{}
		for _, path := range p.Owners[owner] {
			files = append(files, core.GeneratedFile{Path: path})
		}
		if err := budget.Add(owner, core.GenerationArtifacts{Files: files, NameLock: p.NameLocks[owner]}); err != nil {
			return empty, err
		}
	}
	if _, replay := p.Owners[s.Owner]; replay {
		// A replay retains the existing lock rather than double-counting it.
		budget, err = NewSplitAdoptionBudget(p.Files)
		if err != nil {
			return empty, err
		}
	}
	if err := budget.Add(s.Owner, generated); err != nil {
		return empty, err
	}
	original := map[string]string{}
	bytesRetained := 0
	for _, file := range p.Files {
		if _, exists := original[file.Path]; exists || m.Files[file.Path] != digest([]byte(file.Content)) || len(file.Content) > maxAdoptionKotlinBytes-bytesRetained {
			return empty, splitAdoptionChanged()
		}
		bytesRetained += len(file.Content)
		original[file.Path] = file.Content
	}
	if len(original) != len(m.Files) || len(generated.Files) == 0 || generated.NameLock == nil {
		return empty, splitAdoptionChanged()
	}
	donorPaths := map[string]bool{}
	for _, path := range p.Owners[s.Donor] {
		donorPaths[path] = true
	}
	paths := []string{}
	seen := map[string]bool{}
	for _, file := range generated.Files {
		if relativePath(file.Path) != nil || seen[file.Path] || !donorPaths[file.Path] || original[file.Path] != file.Content {
			return empty, splitAdoptionChanged()
		}
		seen[file.Path] = true
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	before, err := parseNameLockBytes(p.NameLocks[s.Donor])
	if err != nil {
		return empty, err
	}
	after, err := parseNameLockBytes(generated.NameLock)
	if err != nil {
		return empty, err
	}
	if before.Profile != after.Profile {
		return empty, fail("E_NAME_LOCK_PROFILE", "Existing-owner adoption requires the explicit donor name-lock profile.")
	}
	// Membership alone cannot authorize a new allocation or occurrence.
	allocations := map[string]int{}
	for i, entry := range before.Entries {
		allocations[entry.Role+":"+entry.Shape] = i
	}
	for _, entry := range after.Entries {
		i, ok := allocations[entry.Role+":"+entry.Shape]
		if !ok || before.Entries[i].Name != entry.Name {
			return empty, splitAdoptionChanged()
		}
		old := before.Entries[i].Occurrences
		for _, occurrence := range entry.Occurrences {
			index := sort.SearchStrings(old, occurrence)
			if index == len(old) || old[index] != occurrence {
				return empty, splitAdoptionChanged()
			}
		}
	}
	payload := generated.NameLock
	if reflect.DeepEqual(before, after) {
		payload = p.NameLocks[s.Donor]
	}
	if existing, ok := p.Owners[s.Owner]; ok {
		old, err := parseNameLockBytes(p.NameLocks[s.Owner])
		if err != nil {
			return empty, err
		}
		if !reflect.DeepEqual(existing, paths) || !reflect.DeepEqual(old, after) {
			return empty, splitAdoptionChanged()
		}
		return SplitLockedAdoptionMetadata{Manifest: p.Manifest, NameLocks: map[string][]byte{s.Owner: p.NameLocks[s.Owner]}}, nil
	}
	if len(p.OwnerNames) >= maxLockedOwners {
		return empty, lockedAdoptionInvalid()
	}
	if len(payload) > maxOwnerNameLockBytes-lockBytes {
		return empty, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
	}
	owners, hashes := map[string][]string{}, map[string]string{}
	for name, paths := range p.Owners {
		owners[name] = paths
	}
	for name, hash := range *m.NameLocks {
		hashes[name] = hash
	}
	owners[s.Owner], hashes[s.Owner] = paths, digest(payload)
	m.Owners, m.NameLocks = &owners, &hashes
	raw, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		return empty, ioFailure()
	}
	if _, err := parseManifestKind(raw, true, true); err != nil {
		return empty, err
	}
	return SplitLockedAdoptionMetadata{Manifest: raw, NameLocks: map[string][]byte{s.Owner: payload}}, nil
}

func WriteExistingLockedAdoption(root string, u ExistingLockedAdoptionUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(map[string]ExistingLockedAdoptionUpdate{root: u}, check, nil, makeExistingLockedAdoptionPlan)
}

func makeExistingLockedAdoptionPlan(root string, u ExistingLockedAdoptionUpdate) (*plan, *diagnostic.Diagnostic) {
	s, p := u.Previous, u.Previous.Previous
	if p.identity == nil || p.Root == "" || p.SourceRoot == "" {
		return nil, lockedAdoptionInvalid()
	}
	bound, err := canonicalRoot(p.SourceRoot)
	if err != nil {
		return nil, err
	}
	stale := func() *diagnostic.Diagnostic {
		return fail("E_LOCKED_OWNER_ADOPTION_STALE", "Original root, ownership, Kotlin or raw locks changed after existing-owner adoption planning.")
	}
	if bound != p.Root || root != bound {
		return nil, stale()
	}
	current, err := ReadExistingLockedAdoptionState(p.SourceRoot, s.Owner, s.Donor)
	if err != nil {
		return nil, err
	}
	c := current.Previous
	if !os.SameFile(c.identity, p.identity) || !bytes.Equal(c.Manifest, p.Manifest) || !reflect.DeepEqual(c.NameLocks, p.NameLocks) || !reflect.DeepEqual(c.Files, p.Files) || !reflect.DeepEqual(c.Owners, p.Owners) || !reflect.DeepEqual(c.OwnerNames, p.OwnerNames) || c.Owner != p.Owner || c.Version != p.Version {
		return nil, stale()
	}
	next, err := PlanExistingLockedAdoption(current, u.Generated)
	if err != nil {
		return nil, err
	}
	plan := &plan{Writes: map[string][]byte{}, ManifestBytes: next.Manifest, Changed: !bytes.Equal(next.Manifest, c.Manifest)}
	if plan.Changed {
		plan.Writes[ownerNameLockFile(s.Owner)] = next.NameLocks[s.Owner]
	}
	return plan, nil
}
