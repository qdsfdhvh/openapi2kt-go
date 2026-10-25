package output

import (
	"bytes"
	"openapi-kt/internal/diagnostic"
	"os"
	"reflect"
)

// Validate detached source authority before planning metadata-only migrations.
// Files and locks must be exactly those from the raw original manifest; this
// does not manufacture ownership or substitute generated artifacts for source.
func validateRootMoveCapture(s RootMoveState, kotlinLimit int, invalid *diagnostic.Diagnostic) (*manifest, *diagnostic.Diagnostic) {
	if s.root == "" || s.MergeTarget != nil || len(s.bindings) != 0 {
		return nil, invalid
	}
	m, e := parseManifestKind(s.Manifest, false, true)
	if e != nil && e.Code == "E_MANIFEST_VERSION" {
		m, e = parseManifestKind(s.Manifest, true, true)
	}
	if e != nil {
		return nil, e
	}
	owners := map[string][]string{}
	if m.Owners != nil {
		owners = *m.Owners
	}
	locks := map[string]string{}
	if m.NameLock != nil {
		locks[""] = *m.NameLock
	}
	if m.NameLocks != nil {
		locks = *m.NameLocks
	}
	entry, captured := s.Entries[manifestName]
	if m.Version != s.Version || !captured || !bytes.Equal(entry.Data, s.Manifest) || !reflect.DeepEqual(owners, s.Owners) || len(s.NameLocks) != len(locks) || len(s.Files) != len(m.Files) || len(s.Entries) != len(m.Files)+len(locks)+1 {
		return nil, invalid
	}
	total := 0
	last := ""
	for _, f := range s.Files {
		entry, captured := s.Entries[f.Path]
		if !captured || f.Path <= last || m.Files[f.Path] != digest([]byte(f.Content)) || string(entry.Data) != f.Content {
			return nil, invalid
		}
		last = f.Path
		if kotlinLimit >= 0 && len(f.Content) > kotlinLimit-total {
			return nil, fail("E_OUTPUT_LIMIT", "Original metadata migration Kotlin exceeds the aggregate source limit.")
		}
		total += len(f.Content)
	}
	total = 0
	for _, key := range sortedKeys(locks) {
		raw, exists := s.NameLocks[key]
		path := ownerNameLockFile(key)
		if key == "" {
			path = nameLockFile
		}
		entry, captured := s.Entries[path]
		if !exists || !captured || locks[key] != digest(raw) || !bytes.Equal(entry.Data, raw) {
			return nil, invalid
		}
		if len(raw) > maxOwnerNameLockBytes-total {
			return nil, fail("E_NAME_LOCK_LIMIT", "Original raw name-lock inventory exceeds metadata limits.")
		}
		total += len(raw)
		if e := validateNameLockBytes(raw); e != nil {
			return nil, e
		}
	}
	return m, nil
}

// Stat through a handle so Windows captures the file ID immediately rather than
// resolving a saved path lazily when os.SameFile is called after a replacement.
func captureRootIdentity(root string) (os.FileInfo, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return info, nil
}

// Resolve once for IO, then recheck the declared binding and physical identity
// before inputs. A changed ancestor cannot redirect the captured file reads.
func readMetadataRootCapture(root string, stale *diagnostic.Diagnostic) (RootMoveState, os.FileInfo, *diagnostic.Diagnostic) {
	canonical, e := canonicalRoot(root)
	if e != nil {
		return RootMoveState{}, nil, e
	}
	identity, err := captureRootIdentity(canonical)
	if err != nil {
		return RootMoveState{}, nil, ioFailure()
	}
	previous, e := readRootMoveState(canonical, maxAdoptionKotlinBytes)
	if e != nil {
		return RootMoveState{}, nil, e
	}
	bound, e := canonicalRoot(root)
	if e != nil {
		return RootMoveState{}, nil, e
	}
	current, err := os.Stat(canonical)
	if err != nil || bound != canonical || previous.root != canonical || !os.SameFile(identity, current) {
		return RootMoveState{}, nil, stale
	}
	return previous, identity, nil
}
