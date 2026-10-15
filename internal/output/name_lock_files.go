package output

import (
	"bytes"
	"path/filepath"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// ReadNameLockFiles returns detached, path-sorted Kotlin files with their verified
// stored contents. An empty owner selects direct output; a nonempty owner selects
// only that named owner's membership, including shared files. Reading actual
// output preserves post-merge typealiases rather than inferring declarations from
// pre-merge name-lock allocations.
// The root, recovery state, manifest and owned lock are validated before exact
// expected-state comparison and file inspection. It never creates directories
// or acquires a lock, and provides a point-in-time read, not a concurrent snapshot.
func ReadNameLockFiles(root, owner string, expectedPrevious []byte) ([]core.GeneratedFile, *diagnostic.Diagnostic) {
	named := owner != ""
	if named && !ownerName(owner) {
		return nil, fail("E_OWNER_UPDATE_INVALID", "Choose an ASCII owner identifier.")
	}
	if e := validateRoot(root); e != nil {
		return nil, e
	}
	if e := inspectRecovery(root); e != nil {
		return nil, e
	}
	previous, e := readManifestKind(root, named, true)
	if named && e != nil && e.Code == "E_MANIFEST_OWNER_MIGRATION" {
		// Match ReadOwnerNameLock: reading legacy output does not adopt it.
		previous, e = readManifest(root)
	}
	if e != nil {
		return nil, e
	}
	var current []byte
	if named {
		var hash *string
		if previous != nil && previous.NameLocks != nil {
			if h, owned := (*previous.NameLocks)[owner]; owned {
				hash = &h
			}
		}
		current, e = readNamedState(root, owner, hash)
	} else {
		current, e = readManagedNameLock(root, previous)
	}
	if e != nil {
		return nil, e
	}
	if current == nil {
		return nil, fail("E_NAME_LOCK_LIFECYCLE", "A name-lock lifecycle action requires existing manifest-owned state.")
	}
	if expectedPrevious == nil || !bytes.Equal(current, expectedPrevious) {
		return nil, fail("E_NAME_LOCK_STALE", "Name-lock state changed after generation; read the current state and generate again.")
	}
	return readOwnedNameLockFiles(root, previous, owner)
}

// readOwnedNameLockFiles shares the strict old-Kotlin guard with lifecycle plans.
// Callers have already checked owned state and CAS against this manifest. Each
// call checks directories afresh, including the lifecycle lock-time replan.
func readOwnedNameLockFiles(root string, previous *manifest, owner string) ([]core.GeneratedFile, *diagnostic.Diagnostic) {
	var paths []string
	if owner != "" {
		paths = (*previous.Owners)[owner]
	} else {
		paths = sortedKeys(previous.Files)
	}
	files := make([]core.GeneratedFile, 0, len(paths))
	checkedParents := map[string]bool{}
	for _, path := range paths {
		if e := inspectParentsWithCache(root, path, checkedParents); e != nil {
			return nil, e
		}
		data, exists, e := readRegular(filepath.Join(root, filepath.FromSlash(path)))
		if e != nil {
			return nil, e
		}
		if !exists {
			return nil, fail("E_OUTPUT_MISSING", "A selected owner's generated file is missing; restore it before migration.")
		}
		hash := previous.Files[path]
		if digest(data) != hash {
			return nil, fail("E_OUTPUT_MODIFIED", "A previously generated file was modified. Restore or relocate it before generation.")
		}
		files = append(files, core.GeneratedFile{Path: path, Content: string(data)})
	}
	return files, nil
}
