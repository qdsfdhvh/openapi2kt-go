package output

import (
	"bytes"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/schema"
	"path/filepath"
)

const maxLockedOwners = 256
const maxOwnerNameLockBytes = 16 << 20

// NameLockedOwnerUpdate supplies exact prior state for generated locked owners
// and selected owners explicitly disabling their lock. Each needs a key, even
// when its prior state is nil. Actions require existing owned state; reset keeps
// its profile, and callers must verify the profile before generating disable
// artifacts, which do not encode a profile.
// Caller inventories, artifacts and byte slices must remain immutable in-flight.
type NameLockedOwnerUpdate struct {
	OwnerUpdate
	PreviousNameLocks map[string][]byte
	NameLockActions   map[string]NameLockAction
}

func ownerNameLockFile(owner string) string {
	return ".openapi-kt-names-" + digest([]byte(owner)) + ".json"
}
func WriteNameLockedOwners(root string, u NameLockedOwnerUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writePlanned(root, check, func(root string) (*plan, *diagnostic.Diagnostic) { return makeNameLockedOwnersPlan(root, u) })
}
func makeNameLockedOwnersPlan(root string, u NameLockedOwnerUpdate) (*plan, *diagnostic.Diagnostic) {
	return makeOwnersPlanMode(root, u.OwnerUpdate, &u.PreviousNameLocks, u.NameLockActions)
}

// WriteNameLockedOwnersBatch uses the same complete lock set and all-prepare /
// reverse rollback transaction engine as ordinary owners, not sidecar writes.
func WriteNameLockedOwnersBatch(updates map[string]NameLockedOwnerUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(updates, check, nil, makeNameLockedOwnersPlan)
}
func validateOwnerNameLocks(u OwnerUpdate, expected map[string][]byte, actions map[string]NameLockAction) *diagnostic.Diagnostic {
	for _, owner := range sortedKeys(actions) {
		artifact, selected := u.Generated[owner]
		if !selected || !validNameLockAction(actions[owner]) {
			return fail("E_OWNER_UPDATE_INVALID", "Name-lock actions require selected owners and an explicit reset or disable action.")
		}
		if e := validateNameLockActionArtifacts(artifact, actions[owner]); e != nil {
			return e
		}
	}
	count, total := 0, 0
	for _, owner := range sortedKeys(u.Generated) {
		artifact := u.Generated[owner]
		if artifact.NameLock == nil {
			if actions[owner] == NameLockDisable {
				if _, ok := expected[owner]; !ok {
					return fail("E_OWNER_UPDATE_INVALID", "Every explicitly disabled owner requires exact previous name-lock state.")
				}
			}
			continue
		}
		if len(artifact.Files) == 0 {
			return fail("E_NAME_LOCK_OUTPUT_INVALID", "A locked owner requires generated Kotlin membership.")
		}
		if _, ok := expected[owner]; !ok {
			return fail("E_OWNER_UPDATE_INVALID", "Every generated locked owner requires exact previous name-lock state.")
		}
		count++
		total += len(artifact.NameLock)
		if count > maxLockedOwners || total > maxOwnerNameLockBytes {
			return fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
		if e := validateNameLockBytes(artifact.NameLock); e != nil {
			return e
		}
	}
	for _, owner := range sortedKeys(expected) {
		if a, ok := u.Generated[owner]; !ok || a.NameLock == nil && actions[owner] != NameLockDisable {
			return fail("E_OWNER_UPDATE_INVALID", "Previous name-lock state is only valid for generated locked or explicitly disabled owners.")
		}
	}
	return nil
}
func readNamedState(root, owner string, hash *string) ([]byte, *diagnostic.Diagnostic) {
	data, exists, e := readRegularLimited(filepath.Join(root, ownerNameLockFile(owner)), schema.MaxNameLockBytes)
	if e != nil {
		return nil, e
	}
	if hash == nil {
		if exists {
			return nil, fail("E_OUTPUT_COLLISION", "Named name-lock metadata is not owned by this manifest.")
		}
		return nil, nil
	}
	if !exists {
		return nil, fail("E_NAME_LOCK_MISSING", "Restore the missing owned name lock before generating again.")
	}
	if digest(data) != *hash {
		return nil, fail("E_NAME_LOCK_MODIFIED", "The owned name lock was modified; restore it before generating again.")
	}
	if e := validateNameLockBytes(data); e != nil {
		return nil, e
	}
	return data, nil
}

// ReadOwnerNameLock reads only the selected payload. A version 1 read does not
// adopt legacy Kotlin ownership; a later write still needs explicit exact-match
// AdoptLegacy. Unsupported single-owner locked version 3 is never guessed.
func ReadOwnerNameLock(root, owner string) ([]byte, *diagnostic.Diagnostic) {
	if !ownerName(owner) {
		return nil, fail("E_OWNER_UPDATE_INVALID", "Choose an ASCII owner identifier.")
	}
	if e := validateRoot(root); e != nil {
		return nil, e
	}
	if e := inspectRecovery(root); e != nil {
		return nil, e
	}
	previous, e := readManifestKind(root, true, true)
	if e != nil && e.Code == "E_MANIFEST_OWNER_MIGRATION" {
		previous, e = readManifest(root)
	}
	if e != nil {
		return nil, e
	}
	var hash *string
	if previous != nil && previous.NameLocks != nil {
		if h, ok := (*previous.NameLocks)[owner]; ok {
			hash = &h
		}
	}
	return readNamedState(root, owner, hash)
}

type ownerNameLockPlan struct {
	hashes   map[string]string
	writes   map[string][]byte
	removals []string
}

func prepareOwnerNameLocks(root string, previous *manifest, u OwnerUpdate, expected map[string][]byte, actions map[string]NameLockAction) (*ownerNameLockPlan, *diagnostic.Diagnostic) {
	p := &ownerNameLockPlan{hashes: map[string]string{}, writes: map[string][]byte{}}
	old := map[string]string{}
	if previous != nil && previous.NameLocks != nil {
		old = *previous.NameLocks
	}
	current := map[string][]byte{}
	total := 0
	// Verify every owned state, including those being removed. Never delete an
	// edited/missing payload merely because its owner was reconciled away.
	for _, owner := range sortedKeys(old) {
		hash := old[owner]
		data, e := readNamedState(root, owner, &hash)
		if e != nil {
			return nil, e
		}
		total += len(data)
		if total > maxOwnerNameLockBytes {
			return nil, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
		current[owner] = data
	}
	active := map[string]bool{}
	for _, owner := range u.ActiveOwners {
		active[owner] = true
	}
	for _, owner := range sortedKeys(old) {
		if _, selected := u.Generated[owner]; selected {
			continue
		}
		if u.Reconcile && !active[owner] {
			p.removals = append(p.removals, ownerNameLockFile(owner))
			continue
		}
		p.hashes[owner] = old[owner]
	}
	for _, owner := range sortedKeys(u.Generated) {
		artifact := u.Generated[owner]
		prior, owned := current[owner]
		action := actions[owner]
		if action != "" && !owned {
			// Inspect even an unowned destination before rejecting lifecycle use,
			// preserving collision and symlink protection on fresh output.
			if _, e := readNamedState(root, owner, nil); e != nil {
				return nil, e
			}
			return nil, fail("E_NAME_LOCK_LIFECYCLE", "A name-lock lifecycle action requires existing manifest-owned state.")
		}
		if artifact.NameLock == nil && action != NameLockDisable {
			if owned {
				return nil, fail("E_NAME_LOCK_DISABLED", "A selected owner has stored name-lock state; choose an explicit disable action.")
			}
			continue
		}
		if !owned {
			var e *diagnostic.Diagnostic
			prior, e = readNamedState(root, owner, nil)
			if e != nil {
				return nil, e
			}
		}
		before := expected[owner]
		if (prior == nil) != (before == nil) || !bytes.Equal(prior, before) {
			return nil, fail("E_NAME_LOCK_STALE", "Name-lock state changed after generation; read the current state and generate again.")
		}
		if action == NameLockReset {
			if e := sameNameLockProfile(prior, artifact.NameLock); e != nil {
				return nil, e
			}
		}
		if action != "" {
			if _, e := readOwnedNameLockFiles(root, previous, owner); e != nil {
				return nil, e
			}
		}
		if action == NameLockDisable {
			p.removals = append(p.removals, ownerNameLockFile(owner))
			continue
		}
		p.hashes[owner] = digest(artifact.NameLock)
		if prior == nil || !bytes.Equal(prior, artifact.NameLock) {
			p.writes[ownerNameLockFile(owner)] = artifact.NameLock
		}
	}
	total = 0
	for _, owner := range sortedKeys(p.hashes) {
		if a, selected := u.Generated[owner]; selected {
			total += len(a.NameLock)
		} else {
			total += len(current[owner])
		}
		if len(p.hashes) > maxLockedOwners || total > maxOwnerNameLockBytes {
			return nil, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
	}
	return p, nil
}
