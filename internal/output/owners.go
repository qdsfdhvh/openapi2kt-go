package output

import (
	"bytes"
	"path/filepath"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// OwnerUpdate replaces selected owners and optionally reconciles owners absent
// from the complete active inventory. It applies within one output root only.
// Callers must not mutate the inventory/artifacts while an update is running.
type OwnerUpdate struct {
	ActiveOwners []string
	Generated    map[string]core.GenerationArtifacts
	Reconcile    bool
	// AdoptLegacy explicitly assigns an exact-match version 1 manifest to this owner.
	AdoptLegacy string
}

func ownerName(name string) bool {
	if len(name) == 0 || !(name[0] >= 'a' && name[0] <= 'z' || name[0] >= 'A' && name[0] <= 'Z') {
		return false
	}
	for _, c := range []byte(name) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func parseOwners(v *document.Value, files map[string]string) (map[string][]string, *diagnostic.Diagnostic) {
	bad := func() (map[string][]string, *diagnostic.Diagnostic) {
		return nil, fail("E_MANIFEST_INVALID", "Named-owner manifest membership is invalid.")
	}
	if v.Kind() != document.Object {
		return bad()
	}
	out := map[string][]string{}
	covered := map[string]bool{}
	for _, member := range v.Members() {
		if !ownerName(member.Name) || member.Value.Kind() != document.Array || len(member.Value.Items()) == 0 {
			return bad()
		}
		last := ""
		var paths []string
		for _, item := range member.Value.Items() {
			path, ok := item.Text()
			if !ok || path <= last || files[path] == "" {
				return bad()
			}
			paths = append(paths, path)
			covered[path] = true
			last = path
		}
		out[member.Name] = paths
	}
	if len(covered) != len(files) {
		return bad()
	}
	return out, nil
}

// WriteOwners commits version 2 ownership within one root. No legacy files are
// adopted; check is read-only, and retained owners are reloaded under the lock.
func WriteOwners(root string, update OwnerUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writePlanned(root, check, func(root string) (*plan, *diagnostic.Diagnostic) { return makeOwnersPlan(root, update) })
}
func makeOwnersPlan(root string, update OwnerUpdate) (*plan, *diagnostic.Diagnostic) {
	return makeOwnersPlanMode(root, update, nil, nil)
}
func makeOwnersPlanMode(root string, update OwnerUpdate, expected *map[string][]byte, actions map[string]NameLockAction) (*plan, *diagnostic.Diagnostic) {
	if expected == nil {
		for _, artifact := range update.Generated {
			if artifact.NameLock != nil {
				return nil, fail("E_NAME_LOCK_OUTPUT_UNSUPPORTED", "This output adapter cannot transactionally persist name-lock metadata.")
			}
		}
	} else if e := validateOwnerNameLocks(update, *expected, actions); e != nil {
		return nil, e
	}
	active := map[string]bool{}
	for _, owner := range update.ActiveOwners {
		if !ownerName(owner) || active[owner] {
			return nil, fail("E_OWNER_UPDATE_INVALID", "Active owners must be distinct ASCII identifiers.")
		}
		active[owner] = true
	}
	for _, owner := range sortedKeys(update.Generated) {
		if !active[owner] {
			return nil, fail("E_OWNER_UPDATE_INVALID", "Generated owners must belong to the active inventory.")
		}
	}
	if len(update.Generated) == 0 && !update.Reconcile {
		return nil, fail("E_OUTPUT_EMPTY", "Select owners or explicitly request reconciliation.")
	}
	// Validate every incoming path before inspecting or mutating ownership.
	selected := map[string]map[string][]byte{}
	for _, owner := range sortedKeys(update.Generated) {
		files := map[string][]byte{}
		used := map[string]bool{}
		for _, file := range update.Generated[owner].Files {
			path := file.Path
			if err := relativePath(path); err != nil {
				return nil, err
			}
			if used[strings.ToLower(path)] {
				return nil, fail("E_OUTPUT_COLLISION", "Generated owner paths collide ignoring case.")
			}
			used[strings.ToLower(path)] = true
			files[path] = []byte(file.Content)
		}
		selected[owner] = files
	}
	if update.AdoptLegacy != "" {
		if _, exists := selected[update.AdoptLegacy]; !exists || len(selected) != 1 {
			return nil, fail("E_OWNER_UPDATE_INVALID", "Legacy adoption requires exactly one generated active owner.")
		}
	}
	if err := inspectRecovery(root); err != nil {
		return nil, err
	}
	previous, err := readOwnerManifestKind(root, update.AdoptLegacy, selected[update.AdoptLegacy], expected != nil)
	if err != nil {
		return nil, err
	}
	var metadata *ownerNameLockPlan
	if expected != nil {
		metadata, err = prepareOwnerNameLocks(root, previous, update, *expected, actions)
		if err != nil {
			return nil, err
		}
	}
	owners := map[string][]string{}
	desired := map[string][]byte{}
	spellings := map[string]string{}
	add := func(path string, data []byte) *diagnostic.Diagnostic {
		lower := strings.ToLower(path)
		if old, ok := spellings[lower]; ok && old != path {
			return fail("E_OUTPUT_COLLISION", "Owner paths collide ignoring case.")
		}
		if old, exists := desired[path]; exists && !bytes.Equal(old, data) {
			return fail("E_OUTPUT_COLLISION", "Shared owner files must have identical content.")
		}
		spellings[lower] = path
		desired[path] = data
		return nil
	}
	if previous != nil {
		for _, owner := range sortedKeys(*previous.Owners) {
			if _, replaced := selected[owner]; replaced || update.Reconcile && !active[owner] {
				continue
			}
			owners[owner] = append([]string(nil), (*previous.Owners)[owner]...)
			for _, path := range owners[owner] {
				if err := inspectParents(root, path); err != nil {
					return nil, err
				}
				data, exists, err := readRegular(filepath.Join(root, filepath.FromSlash(path)))
				if err != nil {
					return nil, err
				}
				if !exists {
					return nil, fail("E_OUTPUT_MISSING", "A retained owner's generated file is missing; regenerate that owner first.")
				}
				if digest(data) != previous.Files[path] {
					return nil, fail("E_OUTPUT_MODIFIED", "A previously generated file was modified. Restore or relocate it before generation.")
				}
				if err := add(path, data); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, owner := range sortedKeys(selected) {
		paths := sortedKeys(selected[owner])
		if len(paths) == 0 {
			continue
		}
		owners[owner] = paths
		for _, path := range paths {
			if err := add(path, selected[owner][path]); err != nil {
				return nil, err
			}
		}
	}
	next := manifest{Version: 2, Files: map[string]string{}, Owners: &owners}
	if metadata != nil && (len(metadata.hashes) != 0 || previous != nil && previous.Version == 4) {
		next.Version = 4
		next.NameLocks = &metadata.hashes
	}
	p, err := makePlanFrom(root, desired, previous, next)
	if err != nil {
		return nil, err
	}
	if metadata != nil {
		for path, data := range metadata.writes {
			p.Writes[path] = data
		}
		p.Removals = append(p.Removals, metadata.removals...)
		p.Changed = p.Changed || len(metadata.writes) != 0 || len(metadata.removals) != 0
	}
	return p, nil
}
