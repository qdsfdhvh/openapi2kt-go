package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"openapi-kt/internal/diagnostic"
)

func rootMoveOccupied() *diagnostic.Diagnostic {
	return fail("E_OUTPUT_ROOT_MOVE_OCCUPIED", "Owned target merge requires same-version v2/v4 named roots, disjoint selected owner identities and identical bytes for any shared Kotlin paths.")
}

// Capture the entire target before any source inputs. Reject owner/version
// conflicts before opening payloads, retaining the old occupied-root boundary.
func bindRootMoveMergeTarget(root string, selected RootMoveState) (*RootMoveState, *diagnostic.Diagnostic) {
	if _, err := os.Lstat(filepath.Join(root, manifestName)); os.IsNotExist(err) {
		return nil, inspectMoveTarget(root, selected)
	} else if err != nil {
		return nil, ioFailure()
	}
	if selected.Version != 2 && selected.Version != 4 {
		return nil, rootMoveOccupied()
	}
	m, e := readManifestKind(root, true, true)
	if e != nil {
		if e.Code == "E_MANIFEST_VERSION" || e.Code == "E_MANIFEST_OWNER_MIGRATION" {
			return nil, rootMoveOccupied()
		}
		return nil, e
	}
	if m.Version != selected.Version {
		return nil, rootMoveOccupied()
	}
	for owner := range selected.Owners {
		if _, exists := (*m.Owners)[owner]; exists {
			return nil, rootMoveOccupied()
		}
	}
	target, e := ReadRootMoveState(root)
	if e != nil {
		return nil, e
	}
	if _, e := mergeRootMoveManifest(selected, target); e != nil {
		return nil, e
	}
	return &target, nil
}

// Merge membership only. Payload bytes and their ownership identities are never
// regenerated, deduplicated or adopted. Both independent source inventories
// were already validated; validate their union against the same persisted caps.
func mergeRootMoveManifest(selected, target RootMoveState) ([]byte, *diagnostic.Diagnostic) {
	if selected.Version != target.Version || (selected.Version != 2 && selected.Version != 4) {
		return nil, rootMoveOccupied()
	}
	owners := map[string][]string{}
	files, locks := map[string]string{}, map[string]string{}
	total := 0
	for _, s := range []RootMoveState{target, selected} {
		for _, owner := range sortedKeys(s.Owners) {
			if _, exists := owners[owner]; exists {
				return nil, rootMoveOccupied()
			}
			owners[owner] = s.Owners[owner]
		}
		for _, f := range s.Files {
			if _, exists := files[f.Path]; exists {
				original, owned := target.Entries[f.Path]
				if !owned || string(original.Data) != f.Content {
					return nil, rootMoveOccupied()
				}
				continue
			}
			files[f.Path] = digest([]byte(f.Content))
		}
		for _, owner := range sortedKeys(s.NameLocks) {
			data := s.NameLocks[owner]
			if len(locks) >= maxLockedOwners || len(data) > maxOwnerNameLockBytes-total {
				return nil, fail("E_NAME_LOCK_LIMIT", "Merged named name-lock inventory exceeds metadata limits.")
			}
			locks[owner] = digest(data)
			total += len(data)
		}
	}
	// Case-folded full-path and file/parent overlaps must refuse on every host,
	// even when the current filesystem could store both spellings.
	paths := map[string]bool{}
	for path := range files {
		lower := strings.ToLower(path)
		if paths[lower] {
			return nil, rootMoveOccupied()
		}
		paths[lower] = true
	}
	for path := range paths {
		for parent := filepath.ToSlash(filepath.Dir(path)); parent != "."; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if paths[parent] {
				return nil, rootMoveOccupied()
			}
		}
	}
	checked := map[string]bool{}
	for _, path := range sortedKeys(selected.Entries) {
		if path == manifestName {
			continue
		}
		if _, shared := target.Entries[path]; shared && strings.HasSuffix(path, ".kt") {
			// The union above has already compared the complete original bytes.
			// Keep this existing target payload and its own attributes in place.
			continue
		}
		if e := inspectParentsWithCache(target.root, path, checked); e != nil {
			return nil, e
		}
		if _, err := os.Lstat(filepath.Join(target.root, filepath.FromSlash(path))); err == nil {
			return nil, fail("E_OUTPUT_COLLISION", "Move target contains an unowned destination; no files were changed.")
		} else if !os.IsNotExist(err) {
			return nil, ioFailure()
		}
	}
	m := manifest{Version: selected.Version, Files: files, Owners: &owners}
	if selected.Version == 4 {
		m.NameLocks = &locks
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, rootMoveInvalid()
	}
	data = append(data, '\n')
	if len(data) > maxFileBytes {
		return nil, fail("E_OUTPUT_LIMIT", "Merged output manifest exceeds the supported size limit.")
	}
	if _, e := parseManifestKind(data, true, true); e != nil {
		return nil, e
	}
	return data, nil
}
