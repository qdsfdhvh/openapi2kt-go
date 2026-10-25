package output

import (
	"encoding/json"
	"openapi-kt/internal/diagnostic"
)

// BindRootMoveOwnerTarget captures the same complete source/CAS binding as a
// root move while preflighting only the selected owners' target payloads.
func BindRootMoveOwnerTarget(s RootMoveState, target, declared string, owners []string) (RootMoveState, *diagnostic.Diagnostic) {
	selected, _, err := splitRootMoveOwners(s, owners)
	if err != nil {
		return RootMoveState{}, err
	}
	bound, err := BindRootMoveTarget(selected, target, declared)
	if err != nil {
		return RootMoveState{}, err
	}
	s.bindings = bound.bindings
	s.MergeTarget = bound.MergeTarget
	return s, nil
}

// MoveOutputOwners splits selected v2/v4 ownership to an empty target. Shared
// payloads remain at source when a retained peer still owns them. Both roots
// stage before apply under the existing complete sorted lock set and rollback.
func MoveOutputOwners(source, target string, previous RootMoveState, owners []string, check bool) (bool, *diagnostic.Diagnostic) {
	if len(owners) == 0 {
		return false, rootMoveInvalid()
	}
	if _, _, err := splitRootMoveOwners(previous, owners); err != nil {
		return false, err
	}
	return moveOutputSelection(source, target, previous, owners, check, nil)
}

func splitRootMoveOwners(s RootMoveState, owners []string) (RootMoveState, RootMoveState, *diagnostic.Diagnostic) {
	if owners == nil {
		return s, RootMoveState{}, nil
	}
	if s.Version != 2 && s.Version != 4 || len(owners) == 0 {
		return RootMoveState{}, RootMoveState{}, rootMoveInvalid()
	}
	selected := map[string]bool{}
	for _, owner := range owners {
		if selected[owner] || s.Owners[owner] == nil {
			return RootMoveState{}, RootMoveState{}, rootMoveInvalid()
		}
		selected[owner] = true
	}
	if len(selected) == len(s.Owners) {
		return s, RootMoveState{}, nil
	}
	build := func(wantSelected bool) (RootMoveState, *diagnostic.Diagnostic) {
		part := RootMoveState{root: s.root, Version: s.Version, Owners: map[string][]string{}, NameLocks: map[string][]byte{}, Entries: map[string]rootMoveEntry{}}
		files, locks := map[string]string{}, map[string]string{}
		for _, owner := range sortedKeys(s.Owners) {
			if selected[owner] != wantSelected {
				continue
			}
			part.Owners[owner] = append([]string(nil), s.Owners[owner]...)
			for _, path := range s.Owners[owner] {
				entry, ok := s.Entries[path]
				if !ok {
					return RootMoveState{}, rootMoveInvalid()
				}
				part.Entries[path] = entry
				files[path] = digest(entry.Data)
			}
			if payload, ok := s.NameLocks[owner]; ok {
				path := ownerNameLockFile(owner)
				entry, exists := s.Entries[path]
				if !exists {
					return RootMoveState{}, rootMoveInvalid()
				}
				part.Entries[path] = entry
				part.NameLocks[owner] = payload
				locks[owner] = digest(payload)
			}
		}
		m := manifest{Version: s.Version, Files: files, Owners: &part.Owners}
		if s.Version == 4 {
			m.NameLocks = &locks
		}
		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return RootMoveState{}, rootMoveInvalid()
		}
		// Split ownership intentionally changes manifest membership only. The raw
		// Kotlin and lock payloads, modes and timestamps remain original.
		data = append(data, '\n')
		part.Manifest = data
		entry := s.Entries[manifestName]
		entry.Data = data
		part.Entries[manifestName] = entry
		for _, f := range s.Files {
			if _, ok := files[f.Path]; ok {
				part.Files = append(part.Files, f)
			}
		}
		if _, err := parseManifestKind(data, true, true); err != nil {
			return RootMoveState{}, err
		}
		return part, nil
	}
	moved, err := build(true)
	if err != nil {
		return RootMoveState{}, RootMoveState{}, err
	}
	retained, err := build(false)
	return moved, retained, err
}
