package output

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// RootMoveState is a detached point-in-time read of one complete manifest-owned
// v1/v2/v3/v4 root. Exact bytes and file attributes bind preflight and locked CAS.
// Owners and lock keys retain their original identity; the direct lock key is "".
// Callers must keep this state immutable while planning or committing a move.
// Internal adapters are not a stable public Go API.
type RootMoveState struct {
	root      string            // Canonical source binding captured before input generation.
	bindings  []rootMoveBinding // Target and optional declared config binding.
	Manifest  []byte
	Version   uint32
	Files     []core.GeneratedFile
	Owners    map[string][]string
	NameLocks map[string][]byte
	Entries   map[string]rootMoveEntry
	// MergeTarget captures complete existing target ownership before input reads.
	// A nil target retains the original empty-destination contract.
	MergeTarget *RootMoveState
}
type rootMoveEntry struct {
	Data       []byte
	Attributes fileAttributes
}

func rootMoveInvalid() *diagnostic.Diagnostic {
	return fail("E_OUTPUT_ROOT_MOVE_INVALID", "Output root move requires complete existing ownership, valid owned metadata and an explicit source and target; foreign metadata and partial inventories are unsupported.")
}

// ReadRootMoveState never acquires locks, creates directories or repairs state.
func ReadRootMoveState(root string) (RootMoveState, *diagnostic.Diagnostic) {
	return readRootMoveState(root, -1)
}

// A nonnegative limit bounds actual cumulative Kotlin reads for metadata-only
// lifecycle actions. Ordinary root moves retain their existing per-file limit.
func readRootMoveState(root string, kotlinLimit int64) (RootMoveState, *diagnostic.Diagnostic) {
	s := RootMoveState{}
	canonical, e := canonicalRoot(root)
	if e != nil {
		return s, e
	}
	root = canonical
	if e := validateRoot(root); e != nil {
		return s, e
	}
	if e := inspectRecovery(root); e != nil {
		return s, e
	}
	data, exists, e := readRegular(filepath.Join(root, manifestName))
	if e != nil {
		return s, e
	}
	if !exists {
		return s, rootMoveInvalid()
	}
	m, e := parseManifestKind(data, false, true)
	if e != nil && e.Code == "E_MANIFEST_VERSION" {
		m, e = parseManifestKind(data, true, true)
	}
	if e != nil {
		return s, e
	}
	s = RootMoveState{root: root, Manifest: data, Version: m.Version, Owners: map[string][]string{}, NameLocks: map[string][]byte{}, Entries: map[string]rootMoveEntry{}}
	if m.Owners != nil {
		s.Owners = *m.Owners
	}
	add := func(path string, data []byte) *diagnostic.Diagnostic {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return ioFailure()
		}
		if !info.Mode().IsRegular() {
			return fail("E_OUTPUT_PATH", "Output entries must be regular files.")
		}
		s.Entries[path] = rootMoveEntry{Data: data, Attributes: fileAttributes{Mode: info.Mode().Perm(), Modified: info.ModTime()}}
		return nil
	}
	if e := add(manifestName, data); e != nil {
		return RootMoveState{}, e
	}
	checked := map[string]bool{}
	for _, path := range sortedKeys(m.Files) {
		if e := inspectParentsWithCache(root, path, checked); e != nil {
			return RootMoveState{}, e
		}
		limit := int64(maxFileBytes)
		if kotlinLimit >= 0 {
			limit = kotlinLimit
		}
		data, exists, e := readRegularLimited(filepath.Join(root, filepath.FromSlash(path)), limit)
		if e != nil {
			return RootMoveState{}, e
		}
		if !exists {
			return RootMoveState{}, fail("E_OUTPUT_MISSING", "Restore missing manifest-owned Kotlin before moving output.")
		}
		if digest(data) != m.Files[path] {
			return RootMoveState{}, fail("E_OUTPUT_MODIFIED", "Restore modified manifest-owned Kotlin before moving output.")
		}
		if kotlinLimit >= 0 {
			kotlinLimit -= int64(len(data))
		}
		if e := add(path, data); e != nil {
			return RootMoveState{}, e
		}
		s.Files = append(s.Files, core.GeneratedFile{Path: path, Content: string(data)})
	}
	if m.Version == 3 {
		data, e := readManagedNameLock(root, m)
		if e != nil {
			return RootMoveState{}, e
		}
		s.NameLocks[""] = data
		if e := add(nameLockFile, data); e != nil {
			return RootMoveState{}, e
		}
	}
	total := 0
	if m.NameLocks != nil {
		for _, owner := range sortedKeys(*m.NameLocks) {
			hash := (*m.NameLocks)[owner]
			data, e := readNamedState(root, owner, &hash)
			if e != nil {
				return RootMoveState{}, e
			}
			total += len(data)
			if total > maxOwnerNameLockBytes {
				return RootMoveState{}, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
			}
			s.NameLocks[owner] = data
			if e := add(ownerNameLockFile(owner), data); e != nil {
				return RootMoveState{}, e
			}
		}
	}
	if e := inspectMoveMetadata(root, s.Entries); e != nil {
		return RootMoveState{}, e
	}
	return s, nil
}

// Inspect only reserved root-level metadata, never former roots or user trees.
// Unowned metadata cannot silently acquire ownership or disappear in a move.
func inspectMoveMetadata(root string, owned map[string]rootMoveEntry) *diagnostic.Diagnostic {
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return ioFailure()
	}
	for _, entry := range entries {
		if strings.HasPrefix(strings.ToLower(entry.Name()), ".openapi-kt-names") {
			if _, ok := owned[entry.Name()]; !ok {
				return rootMoveInvalid()
			}
		}
	}
	return nil
}

func inspectMoveTarget(root string, s RootMoveState) *diagnostic.Diagnostic {
	if e := validateRoot(root); e != nil {
		return e
	}
	if e := inspectRecovery(root); e != nil {
		return e
	}
	if _, err := os.Lstat(filepath.Join(root, manifestName)); err == nil {
		return fail("E_OUTPUT_ROOT_MOVE_OCCUPIED", "Move target already has an output manifest; merging or replacing owned roots is unsupported.")
	} else if !os.IsNotExist(err) {
		return ioFailure()
	}
	if e := inspectMoveMetadata(root, nil); e != nil {
		return e
	}
	checked := map[string]bool{}
	for _, path := range sortedKeys(s.Entries) {
		if e := inspectParentsWithCache(root, path, checked); e != nil {
			return e
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); err == nil {
			return fail("E_OUTPUT_COLLISION", "Move target contains an unowned destination; no files were changed.")
		} else if !os.IsNotExist(err) {
			return ioFailure()
		}
	}
	return nil
}

type rootMoveBinding struct{ raw, canonical string }

// BindRootMoveTarget captures target and config ancestor bindings before input
// generation, and verifies empty or compatible named target ownership.
// declared is empty for direct commands. No filesystem entry is created.
func BindRootMoveTarget(s RootMoveState, target, declared string) (RootMoveState, *diagnostic.Diagnostic) {
	dst, e := canonicalRoot(target)
	if e != nil {
		return RootMoveState{}, e
	}
	s.bindings = []rootMoveBinding{{target, dst}}
	if declared != "" {
		root, e := canonicalRoot(declared)
		if e != nil {
			return RootMoveState{}, e
		}
		if root != s.root && root != dst {
			return RootMoveState{}, rootMoveInvalid()
		}
		s.bindings = append(s.bindings, rootMoveBinding{declared, root})
	}
	s.MergeTarget = nil
	if dst != s.root {
		target, e := bindRootMoveMergeTarget(dst, s)
		if e != nil {
			return RootMoveState{}, e
		}
		s.MergeTarget = target
	}
	return s, nil
}

func checkRootMoveBindings(bindings []rootMoveBinding) *diagnostic.Diagnostic {
	for _, binding := range bindings {
		current, e := canonicalRoot(binding.raw)
		if e != nil {
			return e
		}
		if current != binding.canonical {
			return fail("E_OUTPUT_ROOT_MOVE_STALE", "An output or configured root binding changed after its initial observation.")
		}
	}
	return nil
}

// MoveOutputRoot copies the complete verified root without version adoption,
// re-emission, owner reconciliation or root discovery. Both roots use sorted
// cooperative locks and per-filesystem staging; ordinary failures roll back both.
// It does not provide cross-root crash atomicity or automatic recovery.
func MoveOutputRoot(source, target string, previous RootMoveState, check bool) (bool, *diagnostic.Diagnostic) {
	return moveOutputRoot(source, target, previous, check, nil)
}
func moveOutputRoot(source, target string, previous RootMoveState, check bool, before func(int, string, int, string) error) (bool, *diagnostic.Diagnostic) {
	return moveOutputSelection(source, target, previous, nil, check, before)
}

func moveOutputSelection(source, target string, previous RootMoveState, owners []string, check bool, before func(int, string, int, string) error) (bool, *diagnostic.Diagnostic) {
	if e := checkRootMoveBindings(previous.bindings); e != nil {
		return false, e
	}
	src, e := canonicalRoot(source)
	if e != nil {
		return false, e
	}
	if src != previous.root {
		return false, fail("E_OUTPUT_ROOT_MOVE_STALE", "The source root binding changed after its initial observation.")
	}
	dst, e := canonicalRoot(target)
	if e != nil {
		return false, e
	}
	if len(previous.bindings) != 0 && (previous.bindings[0].raw != target || previous.bindings[0].canonical != dst) {
		return false, fail("E_OUTPUT_ROOT_MOVE_STALE", "The selected target differs from its initial binding.")
	}
	identity := source == target
	if src == dst && !identity {
		return false, fail("E_OUTPUT_ROOTS", "Move source and target alias; use the same explicit root for an identity check.")
	}
	// Validate both sides again inside each planning pass, including under the
	// complete lock set. Staging/application only use these canonical bindings.
	planner := func(root string, _ bool) (*plan, *diagnostic.Diagnostic) {
		if e := checkRootMoveBindings(previous.bindings); e != nil {
			return nil, e
		}
		for _, binding := range []struct{ raw, want string }{{source, src}, {target, dst}} {
			current, e := canonicalRoot(binding.raw)
			if e != nil {
				return nil, e
			}
			if current != binding.want {
				return nil, fail("E_OUTPUT_ROOT_MOVE_STALE", "An output root binding changed after planning.")
			}
		}
		state, e := ReadRootMoveState(src)
		if e != nil {
			return nil, e
		}
		expected := previous
		expected.bindings = nil    // Root bindings are checked separately above.
		expected.MergeTarget = nil // Target state is checked independently below.
		if !bytes.Equal(state.Manifest, expected.Manifest) || !reflect.DeepEqual(state, expected) {
			return nil, fail("E_OUTPUT_ROOT_MOVE_STALE", "Source ownership, code, metadata or file attributes changed after planning; review the current state before moving again.")
		}
		p := &plan{Writes: map[string][]byte{}}
		if identity {
			return p, nil
		}
		selected, retained, e := splitRootMoveOwners(state, owners)
		if e != nil {
			return nil, e
		}
		manifestBytes := selected.Manifest
		if previous.MergeTarget == nil {
			if e := inspectMoveTarget(dst, selected); e != nil {
				return nil, e
			}
		} else {
			targetState, e := ReadRootMoveState(dst)
			if e != nil {
				return nil, e
			}
			if !reflect.DeepEqual(targetState, *previous.MergeTarget) {
				return nil, fail("E_OUTPUT_ROOT_MOVE_STALE", "Target ownership, code, metadata or file attributes changed after planning; review both roots before moving again.")
			}
			manifestBytes, e = mergeRootMoveManifest(selected, targetState)
			if e != nil {
				return nil, e
			}
		}
		p.Changed = true
		if root == src {
			for _, path := range sortedKeys(state.Entries) {
				if path != manifestName {
					if _, keep := retained.Entries[path]; !keep {
						p.Removals = append(p.Removals, path)
					}
				}
			}
			if retained.Entries == nil {
				p.RemoveManifest = true
			} else {
				p.ManifestBytes = retained.Manifest
			}
		} else {
			p.Attributes = map[string]fileAttributes{}
			for _, path := range sortedKeys(selected.Entries) {
				entry := selected.Entries[path]
				if previous.MergeTarget != nil && strings.HasSuffix(path, ".kt") {
					if _, shared := previous.MergeTarget.Entries[path]; shared {
						continue // Retain original target bytes, modes and mtimes.
					}
				}
				if path != manifestName {
					p.Attributes[path] = entry.Attributes
					p.Writes[path] = entry.Data
				}
			}
			if previous.MergeTarget == nil {
				p.Attributes[manifestName] = selected.Entries[manifestName].Attributes
			}
			p.ManifestBytes = manifestBytes
		}
		return p, nil
	}
	updates := map[string]bool{src: true, dst: true}
	return writeOwnersBatchWith(updates, check, before, planner)
}
