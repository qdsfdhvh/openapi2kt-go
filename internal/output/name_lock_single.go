package output

import (
	"bytes"
	"path/filepath"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/schema"
)

const nameLockFile = ".openapi-kt-names.json"

func validContentHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range []byte(hash) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// Validation uses the same strict versioned state decoder as generation. Profile
// fields are only the expected values for that decoder, not a second protocol.
func validateNameLockBytes(data []byte) *diagnostic.Diagnostic {
	_, e := parseNameLockBytes(data)
	return e
}
func parseNameLockBytes(data []byte) (*schema.HoistedNameLock, *diagnostic.Diagnostic) {
	if len(data) > schema.MaxNameLockBytes {
		return nil, fail("E_NAME_LOCK_LIMIT", "Hoisted name lock exceeds metadata limits.")
	}
	v, e := document.ParseObject(data, schema.MaxNameLockBytes)
	if e != nil {
		return nil, fail("E_NAME_LOCK_INVALID", "Hoisted name lock is malformed; no files were changed.")
	}
	var pkg, suffix string
	qualified := false
	if p, ok := v.Get("profile"); ok && p.Kind() == document.Object {
		if x, ok := p.Get("package"); ok {
			pkg, _ = x.Text()
		}
		if x, ok := p.Get("suffix"); ok {
			suffix, _ = x.Text()
		}
		if x, ok := p.Get("qualified"); ok {
			qualified, _ = x.Bool()
		}
	}
	return schema.ParseNameLock(data, pkg, suffix, qualified)
}
func readManagedNameLock(root string, previous *manifest) ([]byte, *diagnostic.Diagnostic) {
	data, exists, e := readRegularLimited(filepath.Join(root, nameLockFile), schema.MaxNameLockBytes)
	if e != nil {
		return nil, e
	}
	owned := previous != nil && previous.Version == 3 && previous.NameLock != nil
	if !owned {
		if exists {
			return nil, fail("E_OUTPUT_COLLISION", "Name-lock metadata is not owned by this manifest.")
		}
		return nil, nil
	}
	if !exists {
		return nil, fail("E_NAME_LOCK_MISSING", "Restore the missing owned name lock before generating again.")
	}
	if digest(data) != *previous.NameLock {
		return nil, fail("E_NAME_LOCK_MODIFIED", "The owned name lock was modified; restore it before generating again.")
	}
	if e := validateNameLockBytes(data); e != nil {
		return nil, e
	}
	return data, nil
}

// ReadNameLock never creates directories or acquires a writer lock. Pass these
// exact bytes to both core generation and WriteNameLocked's expectedPrevious.
func ReadNameLock(root string) ([]byte, *diagnostic.Diagnostic) {
	if e := validateRoot(root); e != nil {
		return nil, e
	}
	if e := inspectRecovery(root); e != nil {
		return nil, e
	}
	previous, e := readManifestKind(root, false, true)
	if e != nil {
		return nil, e
	}
	return readManagedNameLock(root, previous)
}

// WriteNameLocked commits Kotlin, fixed managed metadata and manifest together.
// The expected previous bytes are compared in preflight and again under flock;
// stale generation is rejected, not retried or silently regenerated.
func WriteNameLocked(root string, artifacts core.GenerationArtifacts, expectedPrevious []byte, check bool) (bool, *diagnostic.Diagnostic) {
	return writePlanned(root, check, func(root string) (*plan, *diagnostic.Diagnostic) {
		return makeNameLockedPlan(root, artifacts, expectedPrevious)
	})
}
func makeNameLockedPlan(root string, artifacts core.GenerationArtifacts, expectedPrevious []byte) (*plan, *diagnostic.Diagnostic) {
	return makeNameLockPlan(root, artifacts, expectedPrevious, "")
}
func makeNameLockPlan(root string, artifacts core.GenerationArtifacts, expectedPrevious []byte, action NameLockAction) (*plan, *diagnostic.Diagnostic) {
	if e := validateNameLockActionArtifacts(artifacts, action); e != nil {
		return nil, e
	}
	if action != NameLockDisable {
		if artifacts.NameLock == nil {
			return nil, fail("E_NAME_LOCK_OUTPUT_INVALID", "Managed name-lock output requires generated lock state.")
		}
		if e := validateNameLockBytes(artifacts.NameLock); e != nil {
			return nil, e
		}
	}
	desired, used := map[string][]byte{}, map[string]bool{}
	for _, file := range artifacts.Files {
		path := file.Path
		if e := relativePath(path); e != nil {
			return nil, e
		}
		if used[strings.ToLower(path)] {
			return nil, fail("E_OUTPUT_COLLISION", "Generated paths collide ignoring case.")
		}
		used[strings.ToLower(path)] = true
		desired[path] = []byte(file.Content)
	}
	if len(desired) == 0 {
		return nil, fail("E_OUTPUT_EMPTY", "No model files were generated.")
	}
	if e := inspectRecovery(root); e != nil {
		return nil, e
	}
	previous, e := readManifestKind(root, false, true)
	if e != nil {
		return nil, e
	}
	current, e := readManagedNameLock(root, previous)
	if e != nil {
		return nil, e
	}
	if action != "" && current == nil {
		return nil, fail("E_NAME_LOCK_LIFECYCLE", "A name-lock lifecycle action requires existing manifest-owned state.")
	}
	if (current == nil) != (expectedPrevious == nil) || !bytes.Equal(current, expectedPrevious) {
		return nil, fail("E_NAME_LOCK_STALE", "Name-lock state changed after generation; read the current state and generate again.")
	}
	if action == NameLockReset {
		if e := sameNameLockProfile(current, artifacts.NameLock); e != nil {
			return nil, e
		}
	}
	if action != "" {
		if _, e := readOwnedNameLockFiles(root, previous, ""); e != nil {
			return nil, e
		}
	}
	next := manifest{Version: 1, Files: map[string]string{}}
	if action != NameLockDisable {
		hash := digest(artifacts.NameLock)
		next.Version, next.NameLock = 3, &hash
	}
	p, e := makePlanFrom(root, desired, previous, next)
	if e != nil {
		return nil, e
	}
	if action == NameLockDisable {
		p.Removals = append(p.Removals, nameLockFile)
		p.Changed = true
	} else if current == nil || !bytes.Equal(current, artifacts.NameLock) {
		p.Writes[nameLockFile] = artifacts.NameLock
		p.Changed = true
	}
	return p, nil
}
