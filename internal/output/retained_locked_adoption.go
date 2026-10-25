package output

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/schema"
)

type RetainedLockedAdoptionState struct {
	Previous           RootMoveState
	OwnerNames         []string
	NameLock           []byte // Actual original direct/retained raw payload, not a fake owner.
	retained, declared string
	identity           os.FileInfo
}
type RetainedLockedAdoptionUpdate struct {
	Previous  RetainedLockedAdoptionState
	Generated map[string]core.GenerationArtifacts
}
type RetainedLockedAdoptionMetadata struct {
	Manifest  []byte
	NameLocks map[string][]byte
	Donors    map[string]string // Actual original metadata paths for unchanged copies.
}
type retainedAdoptionSource struct {
	manifest *manifest
	path     string
	raw      []byte
	state    *schema.HoistedNameLock
}

func retainedAdoptionInvalid() *diagnostic.Diagnostic {
	return fail("E_RETAINED_LOCKED_ADOPTION_INVALID", "Retained adoption requires explicit original ownership, an existing locked v3/v4 root and distinct nonempty candidate owners in one root.")
}
func retainedAdoptionChanged() *diagnostic.Diagnostic {
	return fail("E_RETAINED_LOCKED_ADOPTION_CHANGED", "Retained adoption must preserve complete original ownership/raw locks and accept only exact original Kotlin/allocation subsets; existing peers require exact replay.")
}

func ReadRetainedLockedAdoptionState(root, retained string, names []string) (RetainedLockedAdoptionState, *diagnostic.Diagnostic) {
	empty := RetainedLockedAdoptionState{}
	names = append([]string{}, names...)
	sort.Strings(names)
	if !ownerName(retained) || len(names) == 0 || len(names) >= maxLockedOwners {
		return empty, retainedAdoptionInvalid()
	}
	for i, name := range names {
		if !ownerName(name) || name == retained || i > 0 && name == names[i-1] {
			return empty, retainedAdoptionInvalid()
		}
	}
	previous, identity, e := readMetadataRootCapture(root, fail("E_RETAINED_LOCKED_ADOPTION_STALE", "Original root identity or binding changed while observing retained adoption."))
	if e != nil {
		return empty, e
	}
	key := retained
	if previous.Version == 3 {
		key = ""
	}
	s := RetainedLockedAdoptionState{Previous: previous, OwnerNames: names, NameLock: previous.NameLocks[key], retained: retained, declared: root, identity: identity}
	if _, e := retainedAdoptionOriginal(s); e != nil {
		return empty, e
	}
	owners := map[string]bool{retained: true}
	for name := range previous.Owners {
		owners[name] = true
	}
	for _, name := range names {
		owners[name] = true
	}
	if len(owners) > maxLockedOwners {
		return empty, fail("E_NAME_LOCK_LIMIT", "Retained adoption owner inventory exceeds metadata limits.")
	}
	return s, nil
}

func retainedAdoptionOriginal(s RetainedLockedAdoptionState) (retainedAdoptionSource, *diagnostic.Diagnostic) {
	empty := retainedAdoptionSource{}
	m, e := validateRootMoveCapture(s.Previous, maxAdoptionKotlinBytes, retainedAdoptionInvalid())
	if e != nil {
		return empty, e
	}
	if !ownerName(s.retained) || len(s.Previous.Files) == 0 || m.Version != 3 && m.Version != 4 {
		return empty, retainedAdoptionInvalid()
	}
	key, path := "", nameLockFile
	if m.Version == 4 {
		key, path = s.retained, ownerNameLockFile(s.retained)
		paths := make([]string, 0, len(s.Previous.Files))
		for _, f := range s.Previous.Files {
			paths = append(paths, f.Path)
		}
		if !reflect.DeepEqual(s.Previous.Owners[s.retained], paths) {
			return empty, retainedAdoptionInvalid()
		}
	}
	raw, exists := s.Previous.NameLocks[key]
	if !exists || !bytes.Equal(raw, s.NameLock) {
		return empty, retainedAdoptionInvalid()
	}
	state, e := parseNameLockBytes(raw)
	if e != nil {
		return empty, e
	}
	return retainedAdoptionSource{m, path, raw, state}, nil
}

func PlanRetainedLockedAdoption(s RetainedLockedAdoptionState, generated map[string]core.GenerationArtifacts) (RetainedLockedAdoptionMetadata, *diagnostic.Diagnostic) {
	empty := RetainedLockedAdoptionMetadata{}
	original, e := retainedAdoptionOriginal(s)
	if e != nil {
		return empty, e
	}
	if len(s.OwnerNames) == 0 || len(s.OwnerNames) >= maxLockedOwners || !reflect.DeepEqual(s.OwnerNames, sortedKeys(generated)) {
		return empty, retainedAdoptionInvalid()
	}
	owners := map[string][]string{}
	payloads := map[string][]byte{}
	donors := map[string]string{}
	for name, paths := range s.Previous.Owners {
		owners[name] = append([]string{}, paths...)
	}
	if s.Previous.Version == 4 {
		for name, raw := range s.Previous.NameLocks {
			payloads[name] = raw
		}
	}
	retainedPaths := []string{}
	files := map[string]string{}
	code := map[string]string{}
	for _, f := range s.Previous.Files {
		retainedPaths = append(retainedPaths, f.Path)
		files[f.Path] = digest([]byte(f.Content))
		code[f.Path] = f.Content
	}
	owners[s.retained] = retainedPaths
	payloads[s.retained] = original.raw
	donors[s.retained] = original.path
	entries := map[string]schema.NameLockEntry{}
	origins := map[string]map[string]bool{}
	for _, entry := range original.state.Entries {
		key := entry.Role + ":" + entry.Shape
		entries[key] = entry
		origins[key] = map[string]bool{}
		for _, origin := range entry.Occurrences {
			origins[key][origin] = true
		}
	}
	oldStates := map[string]*schema.HoistedNameLock{}
	if s.Previous.Version == 4 {
		for _, owner := range sortedKeys(s.Previous.NameLocks) {
			state, e := parseNameLockBytes(s.Previous.NameLocks[owner])
			if e != nil {
				return empty, e
			}
			oldStates[owner] = state
		}
	}
	for _, name := range s.OwnerNames {
		if !ownerName(name) || name == s.retained {
			return empty, retainedAdoptionInvalid()
		}
		a := generated[name]
		if len(a.Files) == 0 || a.NameLock == nil {
			return empty, retainedAdoptionInvalid()
		}
		state, e := parseNameLockBytes(a.NameLock)
		if e != nil {
			return empty, e
		}
		if state.Profile != original.state.Profile {
			return empty, fail("E_NAME_LOCK_PROFILE", "Retained adoption requires the original direct/retained name-lock profile.")
		}
		for _, entry := range state.Entries {
			key := entry.Role + ":" + entry.Shape
			old, exists := entries[key]
			if !exists || old.Name != entry.Name {
				return empty, retainedAdoptionChanged()
			}
			for _, origin := range entry.Occurrences {
				if !origins[key][origin] {
					return empty, retainedAdoptionChanged()
				}
			}
		}
		paths := []string{}
		seen := map[string]bool{}
		for _, f := range a.Files {
			old, exists := code[f.Path]
			if relativePath(f.Path) != nil || !exists || old != f.Content || seen[f.Path] {
				return empty, retainedAdoptionChanged()
			}
			seen[f.Path] = true
			paths = append(paths, f.Path)
		}
		sort.Strings(paths)
		if prior, exists := s.Previous.Owners[name]; exists {
			raw, locked := s.Previous.NameLocks[name]
			if !locked || !reflect.DeepEqual(prior, paths) {
				return empty, retainedAdoptionChanged()
			}
			if !reflect.DeepEqual(oldStates[name], state) {
				return empty, retainedAdoptionChanged()
			}
			owners[name] = paths
			payloads[name] = raw
			donors[name] = ownerNameLockFile(name)
			continue
		}
		payload, e := state.Encode()
		if e != nil {
			return empty, e
		}
		if reflect.DeepEqual(original.state, state) {
			payload = original.raw
			donors[name] = original.path
		} else if s.Previous.Version == 4 {
			for _, source := range sortedKeys(s.Previous.NameLocks) {
				raw := s.Previous.NameLocks[source]
				if reflect.DeepEqual(oldStates[source], state) {
					payload = raw
					donors[name] = ownerNameLockFile(source)
					break
				}
			}
		}
		owners[name] = paths
		payloads[name] = payload
	}
	if len(owners) > maxLockedOwners {
		return empty, fail("E_NAME_LOCK_LIMIT", "Retained adoption owner inventory exceeds metadata limits.")
	}
	budget, e := NewSplitAdoptionBudget(s.Previous.Files)
	if e != nil {
		return empty, e
	}
	for _, name := range sortedKeys(owners) {
		artifact := core.GenerationArtifacts{NameLock: payloads[name]}
		if name == s.retained {
			artifact.Files = s.Previous.Files
		} else if candidate, generatedOwner := generated[name]; generatedOwner {
			artifact.Files = candidate.Files
		} else {
			for _, path := range owners[name] {
				artifact.Files = append(artifact.Files, core.GeneratedFile{Path: path})
			}
		}
		if e := budget.Add(name, artifact); e != nil {
			return empty, e
		}
	}
	hashes := map[string]string{}
	for name, raw := range payloads {
		hashes[name] = digest(raw)
	}
	next := manifest{Version: 4, Files: files, Owners: &owners, NameLocks: &hashes}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return empty, ioFailure()
	}
	data = append(data, '\n')
	if len(data) > maxFileBytes {
		return empty, fail("E_OUTPUT_LIMIT", "Retained adoption ownership metadata exceeds the supported size limit.")
	}
	if _, e := parseManifestKind(data, true, true); e != nil {
		return empty, e
	}
	if reflect.DeepEqual(original.manifest, &next) {
		data = s.Previous.Manifest
	}
	return RetainedLockedAdoptionMetadata{data, payloads, donors}, nil
}

func WriteRetainedLockedAdoption(root string, u RetainedLockedAdoptionUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(map[string]RetainedLockedAdoptionUpdate{root: u}, check, nil, makeRetainedLockedAdoptionPlan)
}
func makeRetainedLockedAdoptionPlan(root string, u RetainedLockedAdoptionUpdate) (*plan, *diagnostic.Diagnostic) {
	s := u.Previous
	stale := func() *diagnostic.Diagnostic {
		return fail("E_RETAINED_LOCKED_ADOPTION_STALE", "Original root identity/binding, ownership, code, raw locks or attributes changed after retained-adoption planning.")
	}
	if s.identity == nil || s.declared == "" {
		return nil, retainedAdoptionInvalid()
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
	next, e := PlanRetainedLockedAdoption(s, u.Generated)
	if e != nil {
		return nil, e
	}
	p := &plan{Writes: map[string][]byte{}, Attributes: map[string]fileAttributes{}, ManifestBytes: next.Manifest, Changed: !bytes.Equal(next.Manifest, current.Manifest)}
	for _, name := range sortedKeys(next.NameLocks) {
		raw := next.NameLocks[name]
		if old, exists := current.NameLocks[name]; current.Version == 4 && exists && bytes.Equal(old, raw) {
			continue
		}
		path := ownerNameLockFile(name)
		p.Writes[path] = raw
		p.Changed = true
		if donor, copied := next.Donors[name]; copied {
			p.Attributes[path] = current.Entries[donor].Attributes
		}
	}
	if current.Version == 3 {
		p.Removals = []string{nameLockFile}
		p.Changed = true
	}
	return p, nil
}
