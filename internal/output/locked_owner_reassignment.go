package output

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/schema"
)

// LockedOwnerReassignmentState binds a complete existing locked v4 root and
// its desired new owner inventory before inputs. Captures remain immutable.
type LockedOwnerReassignmentState struct {
	Previous   RootMoveState
	OwnerNames []string
	NameLock   []byte // Bounded, unambiguous original allocation/occurrence union.
	declared   string
	identity   os.FileInfo
}
type LockedOwnerReassignmentMetadata struct {
	Manifest  []byte
	NameLocks map[string][]byte
	Donors    map[string]string // Original raw payload authority, when unchanged.
}
type LockedOwnerReassignmentUpdate struct {
	Previous  LockedOwnerReassignmentState
	Generated map[string]core.GenerationArtifacts
}

func reassignmentInvalid() *diagnostic.Diagnostic {
	return fail("E_LOCKED_OWNER_REASSIGN_INVALID", "Locked owner reassignment requires a complete existing fully locked version4 inventory, compatible original profiles and an explicit complete desired owner inventory.")
}
func reassignmentChanged() *diagnostic.Diagnostic {
	return fail("E_LOCKED_OWNER_REASSIGN_CHANGED", "Locked owner reassignment must preserve the complete original Kotlin bytes, public allocation names and allocation-occurrence union.")
}

func ReadLockedOwnerReassignmentState(root string, owners []string) (LockedOwnerReassignmentState, *diagnostic.Diagnostic) {
	empty := LockedOwnerReassignmentState{}
	names := append([]string(nil), owners...)
	sort.Strings(names)
	if len(names) == 0 || len(names) > maxLockedOwners {
		return empty, reassignmentInvalid()
	}
	for i, name := range names {
		if !ownerName(name) || i > 0 && name == names[i-1] {
			return empty, reassignmentInvalid()
		}
	}
	previous, identity, e := readMetadataRootCapture(root, fail("E_LOCKED_OWNER_REASSIGN_STALE", "Original output root identity or binding changed while observing ownership."))
	if e != nil {
		return empty, e
	}
	anchor, e := reassignmentUnion(previous)
	if e != nil {
		return empty, e
	}
	return LockedOwnerReassignmentState{Previous: previous, OwnerNames: names, NameLock: anchor, declared: root, identity: identity}, nil
}

// Rebuild original authority from all owned raw locks, never candidate names.
// A union must itself fit the existing single-lock limits before input reads.
func reassignmentUnion(s RootMoveState) ([]byte, *diagnostic.Diagnostic) {
	if s.Version != 4 || s.root == "" || s.MergeTarget != nil || len(s.bindings) != 0 || len(s.Owners) == 0 || len(s.Owners) > maxLockedOwners || len(s.Owners) != len(s.NameLocks) {
		return nil, reassignmentInvalid()
	}
	m, e := parseManifestKind(s.Manifest, true, true)
	if e != nil {
		return nil, e
	}
	if m.Version != 4 || m.Owners == nil || m.NameLocks == nil || !reflect.DeepEqual(s.Owners, *m.Owners) || len(*m.NameLocks) != len(s.NameLocks) || len(s.Entries) != len(m.Files)+len(s.NameLocks)+1 || !bytes.Equal(s.Entries[manifestName].Data, s.Manifest) || len(s.Files) != len(m.Files) || len(s.Files) == 0 {
		return nil, reassignmentInvalid()
	}
	seen := map[string]bool{}
	sourceBytes := 0
	for _, f := range s.Files {
		entry, captured := s.Entries[f.Path]
		if !captured || seen[f.Path] || m.Files[f.Path] != digest([]byte(f.Content)) || string(entry.Data) != f.Content || len(f.Content) > maxAdoptionKotlinBytes-sourceBytes {
			return nil, reassignmentChanged()
		}
		seen[f.Path] = true
		sourceBytes += len(f.Content)
	}
	union := &schema.HoistedNameLock{Version: 1, Entries: []schema.NameLockEntry{}}
	entries := map[string]schema.NameLockEntry{}
	origins := map[string]map[string]bool{}
	names, roles := map[string]string{}, map[string]string{}
	lockBytes, occurrences := 0, 0
	for i, owner := range sortedKeys(s.Owners) {
		raw, ok := s.NameLocks[owner]
		if !ok || (*m.NameLocks)[owner] != digest(raw) || !bytes.Equal(s.Entries[ownerNameLockFile(owner)].Data, raw) {
			return nil, reassignmentInvalid()
		}
		if len(raw) > maxOwnerNameLockBytes-lockBytes {
			return nil, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
		lockBytes += len(raw)
		state, e := parseNameLockBytes(raw)
		if e != nil {
			return nil, e
		}
		if i == 0 {
			union.Profile = state.Profile
		} else if union.Profile != state.Profile {
			return nil, fail("E_NAME_LOCK_PROFILE", "Locked owner reassignment requires one compatible original name-lock profile.")
		}
		for _, entry := range state.Entries {
			key := entry.Role + ":" + entry.Shape
			name := strings.ToLower(entry.Name)
			if old, ok := entries[key]; ok && old.Name != entry.Name || names[name] != "" && names[name] != key {
				return nil, fail("E_NAME_LOCK_CONFLICT", "Original owner allocation names are ambiguous across the complete inventory.")
			}
			names[name] = key
			if origins[key] == nil {
				if len(entries) >= schema.MaxNameLockEntries {
					return nil, fail("E_NAME_LOCK_LIMIT", "Original allocation union exceeds name-lock limits.")
				}
				origins[key] = map[string]bool{}
				entries[key] = entry
			}
			for _, origin := range entry.Occurrences {
				identity := entry.Role + ":" + origin
				if roles[identity] != "" && roles[identity] != key {
					return nil, fail("E_NAME_LOCK_CONFLICT", "Original owner allocation occurrences are ambiguous across the complete inventory.")
				}
				roles[identity] = key
				if !origins[key][origin] {
					if occurrences >= schema.MaxNameLockOccurrences {
						return nil, fail("E_NAME_LOCK_LIMIT", "Original occurrence union exceeds name-lock limits.")
					}
					occurrences++
					origins[key][origin] = true
				}
			}
		}
	}
	for _, key := range sortedKeys(entries) {
		entry := entries[key]
		entry.Occurrences = sortedKeys(origins[key])
		union.Entries = append(union.Entries, entry)
	}
	return union.Encode()
}

func PlanLockedOwnerReassignment(s LockedOwnerReassignmentState, generated map[string]core.GenerationArtifacts) (LockedOwnerReassignmentMetadata, *diagnostic.Diagnostic) {
	empty := LockedOwnerReassignmentMetadata{}
	anchor, e := reassignmentUnion(s.Previous)
	if e != nil {
		return empty, e
	}
	if !bytes.Equal(anchor, s.NameLock) || len(s.OwnerNames) == 0 || len(s.OwnerNames) > maxLockedOwners || !reflect.DeepEqual(s.OwnerNames, sortedKeys(generated)) {
		return empty, reassignmentInvalid()
	}
	original, e := parseNameLockBytes(anchor)
	if e != nil {
		return empty, e
	}
	entries := map[string]schema.NameLockEntry{}
	origins := map[string]map[string]bool{}
	for _, entry := range original.Entries {
		key := entry.Role + ":" + entry.Shape
		entries[key] = entry
		origins[key] = map[string]bool{}
		for _, origin := range entry.Occurrences {
			origins[key][origin] = true
		}
	}
	oldFiles := map[string]string{}
	for _, f := range s.Previous.Files {
		oldFiles[f.Path] = f.Content
	}
	oldStates := map[string]*schema.HoistedNameLock{}
	oldNames := sortedKeys(s.Previous.NameLocks)
	for _, name := range oldNames {
		state, e := parseNameLockBytes(s.Previous.NameLocks[name])
		if e != nil {
			return empty, e
		}
		oldStates[name] = state
	}
	budget, e := NewSplitAdoptionBudget(s.Previous.Files)
	if e != nil {
		return empty, e
	}
	owners := map[string][]string{}
	files, hashes := map[string]string{}, map[string]string{}
	payloads := map[string][]byte{}
	donors := map[string]string{}
	coverage := map[string]map[string]bool{}
	lockBytes := 0
	for _, name := range s.OwnerNames {
		a := generated[name]
		if !ownerName(name) || len(a.Files) == 0 || a.NameLock == nil {
			return empty, reassignmentInvalid()
		}
		if e := budget.Add(name, a); e != nil {
			return empty, e
		}
		candidate, e := parseNameLockBytes(a.NameLock)
		if e != nil {
			return empty, e
		}
		if candidate.Profile != original.Profile {
			return empty, fail("E_NAME_LOCK_PROFILE", "Locked owner reassignment must preserve the original profile.")
		}
		for _, entry := range candidate.Entries {
			key := entry.Role + ":" + entry.Shape
			old, exists := entries[key]
			if !exists || old.Name != entry.Name {
				return empty, reassignmentChanged()
			}
			if coverage[key] == nil {
				coverage[key] = map[string]bool{}
			}
			for _, origin := range entry.Occurrences {
				if !origins[key][origin] {
					return empty, reassignmentChanged()
				}
				coverage[key][origin] = true
			}
		}
		paths := []string{}
		seen := map[string]bool{}
		for _, f := range a.Files {
			old, exists := oldFiles[f.Path]
			if relativePath(f.Path) != nil || seen[f.Path] || !exists || old != f.Content {
				return empty, reassignmentChanged()
			}
			seen[f.Path] = true
			paths = append(paths, f.Path)
			files[f.Path] = digest([]byte(f.Content))
		}
		sort.Strings(paths)
		owners[name] = paths
		payload := a.NameLock
		candidates := append([]string{name}, oldNames...)
		for _, donor := range candidates {
			if state, exists := oldStates[donor]; exists && reflect.DeepEqual(state, candidate) {
				payload = s.Previous.NameLocks[donor]
				donors[name] = donor
				break
			}
		}
		if len(payload) > maxOwnerNameLockBytes-lockBytes {
			return empty, fail("E_NAME_LOCK_LIMIT", "Reassigned raw lock inventory exceeds metadata limits.")
		}
		lockBytes += len(payload)
		payloads[name] = payload
		hashes[name] = digest(payload)
	}
	if len(files) != len(oldFiles) || len(coverage) != len(origins) {
		return empty, reassignmentChanged()
	}
	for key, want := range origins {
		if !reflect.DeepEqual(want, coverage[key]) {
			return empty, reassignmentChanged()
		}
	}
	next := manifest{Version: 4, Files: files, Owners: &owners, NameLocks: &hashes}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return empty, ioFailure()
	}
	data = append(data, '\n')
	if len(data) > maxFileBytes {
		return empty, fail("E_OUTPUT_LIMIT", "Reassigned ownership metadata exceeds the supported size limit.")
	}
	if _, e := parseManifestKind(data, true, true); e != nil {
		return empty, e
	}
	old, e := parseManifestKind(s.Previous.Manifest, true, true)
	if e != nil {
		return empty, e
	}
	if reflect.DeepEqual(old, &next) {
		data = s.Previous.Manifest
	}
	return LockedOwnerReassignmentMetadata{Manifest: data, NameLocks: payloads, Donors: donors}, nil
}

func WriteLockedOwnerReassignment(root string, u LockedOwnerReassignmentUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(map[string]LockedOwnerReassignmentUpdate{root: u}, check, nil, makeLockedOwnerReassignmentPlan)
}
func makeLockedOwnerReassignmentPlan(root string, u LockedOwnerReassignmentUpdate) (*plan, *diagnostic.Diagnostic) {
	s := u.Previous
	stale := func() *diagnostic.Diagnostic {
		return fail("E_LOCKED_OWNER_REASSIGN_STALE", "Original root identity, binding, ownership, Kotlin, raw locks or file attributes changed after reassignment planning.")
	}
	if s.identity == nil || s.declared == "" {
		return nil, reassignmentInvalid()
	}
	canonical, e := canonicalRoot(s.declared)
	if e != nil {
		return nil, e
	}
	if canonical != root || canonical != s.Previous.root {
		return nil, stale()
	}
	identity, err := os.Stat(root)
	if err != nil || !os.SameFile(identity, s.identity) {
		return nil, stale()
	}
	current, e := readRootMoveState(root, maxAdoptionKotlinBytes)
	if e != nil {
		return nil, e
	}
	if !reflect.DeepEqual(current, s.Previous) {
		return nil, stale()
	}
	next, e := PlanLockedOwnerReassignment(s, u.Generated)
	if e != nil {
		return nil, e
	}
	p := &plan{Writes: map[string][]byte{}, Attributes: map[string]fileAttributes{}, ManifestBytes: next.Manifest, Changed: !bytes.Equal(next.Manifest, current.Manifest)}
	for _, name := range sortedKeys(next.NameLocks) {
		payload := next.NameLocks[name]
		if old, exists := current.NameLocks[name]; exists && bytes.Equal(old, payload) {
			continue
		}
		path := ownerNameLockFile(name)
		p.Writes[path] = payload
		p.Changed = true
		if donor, ok := next.Donors[name]; ok {
			p.Attributes[path] = current.Entries[ownerNameLockFile(donor)].Attributes
		}
	}
	for _, name := range sortedKeys(current.NameLocks) {
		if _, keep := next.NameLocks[name]; !keep {
			p.Removals = append(p.Removals, ownerNameLockFile(name))
			p.Changed = true
		}
	}
	return p, nil
}
