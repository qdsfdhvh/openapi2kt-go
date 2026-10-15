// Package output owns filesystem mutation separately from pure generation.
package output

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

const manifestName = ".openapi-kt-manifest.json"
const lockName = ".openapi-kt.lock"
const stagePrefix = ".openapi-kt-stage-"
const maxFileBytes = 64 * 1024 * 1024

type manifest struct {
	Version uint32               `json:"version"`
	Files   map[string]string    `json:"files"`
	Owners  *map[string][]string `json:"owners,omitempty"`
	// Version 3 claims only the fixed single-owner name-lock metadata file.
	NameLock *string `json:"name_lock,omitempty"`
	// Version 4 maps named owners to their dedicated metadata content hashes.
	NameLocks *map[string]string `json:"name_locks,omitempty"`
}
type fileAttributes struct {
	Mode     os.FileMode
	Modified time.Time
}

type plan struct {
	// Root moves preserve the exact manifest and file attributes, or remove the
	// source manifest last. Ordinary generation keeps its existing defaults.
	ManifestBytes  []byte
	RemoveManifest bool
	Attributes     map[string]fileAttributes
	Writes         map[string][]byte
	Removals       []string
	Manifest       manifest
	Changed        bool
}

func fail(code, message string) *diagnostic.Diagnostic { return diagnostic.New(code, message) }
func ioFailure() *diagnostic.Diagnostic {
	return fail("E_OUTPUT_IO", "Cannot access the output files.")
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func relativePath(path string) *diagnostic.Diagnostic {
	bad := !strings.HasSuffix(path, ".kt") || strings.ContainsAny(path, `\:`) || filepath.IsAbs(path)
	for _, c := range []byte(path) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("/_-.", rune(c))) {
			bad = true
		}
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || strings.HasPrefix(part, ".") {
			bad = true
		}
	}
	if bad {
		return fail("E_OUTPUT_PATH", "Manifest and generated paths must be safe relative Kotlin file paths.")
	}
	return nil
}
func readRegular(path string) ([]byte, bool, *diagnostic.Diagnostic) {
	return readRegularLimited(path, maxFileBytes)
}
func readRegularLimited(path string, limit int64) ([]byte, bool, *diagnostic.Diagnostic) {
	meta, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, ioFailure()
	}
	if !meta.Mode().IsRegular() {
		return nil, false, fail("E_OUTPUT_PATH", "Output entries must be regular files, not directories or symbolic links.")
	}
	if meta.Size() > limit {
		return nil, false, fail("E_OUTPUT_LIMIT", "Output metadata or file exceeds the supported size limit.")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false, ioFailure()
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, false, ioFailure()
	}
	if int64(len(data)) > limit {
		return nil, false, fail("E_OUTPUT_LIMIT", "Output file exceeds the supported size limit.")
	}
	return data, true, nil
}
func inspectParents(root, relative string) *diagnostic.Diagnostic {
	return inspectParentsWithCache(root, relative, nil)
}

// checked belongs to one read-only planning pass: true means a real directory,
// false means an absent ancestor. A nil cache keeps standalone checks fresh.
func inspectParentsWithCache(root, relative string, checked map[string]bool) *diagnostic.Diagnostic {
	parts := strings.Split(relative, "/")
	current := root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		if directory, ok := checked[current]; ok {
			if !directory {
				break
			}
			continue
		}
		meta, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if checked != nil {
				checked[current] = false
			}
			break
		}
		if err != nil {
			return ioFailure()
		}
		if !meta.IsDir() || meta.Mode()&os.ModeSymlink != 0 {
			return fail("E_OUTPUT_PATH", "An output parent is not a real directory.")
		}
		if checked != nil {
			checked[current] = true
		}
	}
	return nil
}
func readManifest(root string) (*manifest, *diagnostic.Diagnostic) {
	return readManifestMode(root, false)
}

func readManifestMode(root string, named bool) (*manifest, *diagnostic.Diagnostic) {
	return readManifestKind(root, named, false)
}
func readManifestKind(root string, named, nameLocked bool) (*manifest, *diagnostic.Diagnostic) {
	data, exists, err := readRegular(filepath.Join(root, manifestName))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return parseManifestKind(data, named, nameLocked)
}

// Parse the same bytes that readers use as an expected-state precondition.
func parseManifestKind(data []byte, named, nameLocked bool) (*manifest, *diagnostic.Diagnostic) {
	invalid := func() (*manifest, *diagnostic.Diagnostic) {
		return nil, fail("E_MANIFEST_INVALID", "Output manifest is malformed; no files were changed.")
	}
	value, parseErr := document.ParseObject(data, maxFileBytes)
	if parseErr != nil {
		return invalid()
	}
	version, hasVersion := value.Get("version")
	files, hasFiles := value.Get("files")
	ownerValue, hasOwners := value.Get("owners")
	nameLockValue, hasNameLock := value.Get("name_lock")
	nameLocksValue, hasNameLocks := value.Get("name_locks")
	versionToken, _ := version.Numeric()
	if len(value.Members()) != 2 && !(len(value.Members()) == 3 && hasOwners && string(versionToken) == "2") && !(len(value.Members()) == 3 && hasNameLock && string(versionToken) == "3") && !(len(value.Members()) == 4 && hasOwners && hasNameLocks && string(versionToken) == "4") {
		return invalid()
	}
	if !hasVersion || !hasFiles || files.Kind() != document.Object {
		return invalid()
	}
	token, numeric := version.Numeric()
	if !numeric {
		return invalid()
	}
	number, e := strconv.ParseUint(string(token), 10, 32)
	if e != nil {
		return invalid()
	}
	out := &manifest{Version: uint32(number), Files: map[string]string{}}
	// Validate paths in sorted order, independent of manifest property order.
	entries := append([]document.Member(nil), files.Members()...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	for _, entry := range entries {
		hash, ok := entry.Value.Text()
		if !ok {
			return invalid()
		}
		out.Files[entry.Name] = hash
	}
	if hasNameLock {
		hash, ok := nameLockValue.Text()
		if !ok || !validContentHash(hash) {
			return invalid()
		}
		out.NameLock = &hash
	}
	if hasNameLocks {
		if nameLocksValue.Kind() != document.Object {
			return invalid()
		}
		locks := map[string]string{}
		for _, m := range nameLocksValue.Members() {
			hash, ok := m.Value.Text()
			if !ownerName(m.Name) || !ok || !validContentHash(hash) {
				return invalid()
			}
			locks[m.Name] = hash
		}
		if len(locks) > maxLockedOwners {
			return nil, fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
		}
		out.NameLocks = &locks
	}
	if nameLocked && (!named && number == 3 && !hasNameLock || named && number == 4 && (!hasNameLocks || !hasOwners)) {
		return invalid()
	}
	// Decode every typed slot before interpreting the version, matching the
	// accepted malformed-versus-unsupported-version diagnostic precedence.
	if number != 1 && !(named && number == 2) && !(nameLocked && !named && number == 3) && !(nameLocked && named && number == 4) {
		return nil, fail("E_MANIFEST_VERSION", "Output manifest version is unsupported.")
	}
	used := map[string]bool{}
	for _, path := range sortedKeys(out.Files) {
		if err := relativePath(path); err != nil {
			return nil, err
		}
		hash := out.Files[path]
		valid := len(hash) == 64
		for _, c := range []byte(hash) {
			valid = valid && (c >= '0' && c <= '9' || c >= 'a' && c <= 'f')
		}
		if used[strings.ToLower(path)] || !valid {
			return nil, fail("E_MANIFEST_INVALID", "Manifest paths or content hashes are invalid.")
		}
		used[strings.ToLower(path)] = true
	}
	if named {
		if number == 1 {
			return nil, fail("E_MANIFEST_OWNER_MIGRATION", "Legacy single-owner output requires explicit migration before named-owner updates.")
		}
		if !hasOwners {
			return invalid()
		}
		owners, err := parseOwners(ownerValue, out.Files)
		if err != nil {
			return nil, err
		}
		out.Owners = &owners
		if out.NameLocks != nil {
			for owner := range *out.NameLocks {
				if len(owners[owner]) == 0 {
					return invalid()
				}
			}
		}
	}
	return out, nil
}

func inspectRecovery(root string) *diagnostic.Diagnostic {
	entries, e := os.ReadDir(root)
	if e != nil && !os.IsNotExist(e) {
		return ioFailure()
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagePrefix) {
			return fail("E_RECOVERY_REQUIRED", "An unfinished output transaction exists. Inspect its backups before generating again.")
		}
	}
	return nil
}

func makePlan(root string, desired map[string][]byte) (*plan, *diagnostic.Diagnostic) {
	if err := inspectRecovery(root); err != nil {
		return nil, err
	}
	previous, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	return makePlanFrom(root, desired, previous, manifest{Version: 1, Files: map[string]string{}})
}

func makePlanFrom(root string, desired map[string][]byte, previous *manifest, next manifest) (*plan, *diagnostic.Diagnostic) {
	owned := map[string]string{}
	if previous != nil {
		owned = previous.Files
	}
	// Never share observations with another plan, including the lock-time replan.
	checkedParents := map[string]bool{}
	current := map[string][]byte{}
	for _, path := range sortedKeys(owned) {
		if err := inspectParentsWithCache(root, path, checkedParents); err != nil {
			return nil, err
		}
		data, exists, err := readRegular(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		if exists && digest(data) != owned[path] {
			return nil, fail("E_OUTPUT_MODIFIED", "A previously generated file was modified. Restore or relocate it before generation.")
		}
		current[path] = data
	}
	result := &plan{Writes: map[string][]byte{}, Manifest: next}
	var err *diagnostic.Diagnostic
	for _, path := range sortedKeys(desired) {
		if err := inspectParentsWithCache(root, path, checkedParents); err != nil {
			return nil, err
		}
		actual, wasOwned := current[path]
		if !wasOwned {
			var exists bool
			actual, exists, err = readRegular(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, fail("E_OUTPUT_COLLISION", "A generated destination is not owned by this manifest.")
			}
		}
		// nil marks a missing file; an existing empty file is still present.
		if actual == nil || !bytes.Equal(actual, desired[path]) {
			result.Writes[path] = desired[path]
		}
		result.Manifest.Files[path] = digest(desired[path])
	}
	for _, path := range sortedKeys(owned) {
		if _, present := desired[path]; !present {
			result.Removals = append(result.Removals, path)
		}
	}
	result.Changed = len(result.Writes) > 0 || len(result.Removals) > 0 || previous == nil || !reflect.DeepEqual(*previous, result.Manifest)
	return result, nil
}

// Write returns a safe difference result; check never acquires a lock or mutates
// any output entry. Locks coordinate cooperative writers, not hostile actors.
func Write(root string, artifacts core.GenerationArtifacts, check bool) (bool, *diagnostic.Diagnostic) {
	if artifacts.NameLock != nil {
		return false, fail("E_NAME_LOCK_OUTPUT_UNSUPPORTED", "This output adapter cannot transactionally persist name-lock metadata.")
	}
	return writePlanned(root, check, func(root string) (*plan, *diagnostic.Diagnostic) {
		desired := map[string][]byte{}
		used := map[string]bool{}
		for _, file := range artifacts.Files {
			path := file.Path
			if err := relativePath(path); err != nil {
				return nil, err
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
		return makePlan(root, desired)
	})
}

// The planner runs again while holding the cooperative lock. Named-owner plans
// must reread retained owners rather than commit a stale pre-lock union.
func validateRoot(root string) *diagnostic.Diagnostic {
	badRoot := root == ""
	for _, part := range strings.Split(filepath.ToSlash(root), "/") {
		badRoot = badRoot || part == ".."
	}
	if badRoot {
		return fail("E_OUTPUT_PATH", "Choose an output directory without parent traversal components.")
	}
	meta, e := os.Lstat(root)
	if e == nil && (!meta.IsDir() || meta.Mode()&os.ModeSymlink != 0) {
		return fail("E_OUTPUT_PATH", "Output root must be a real directory.")
	}
	if e != nil && !os.IsNotExist(e) {
		return ioFailure()
	}
	return nil
}

func writePlanned(root string, check bool, planner func(string) (*plan, *diagnostic.Diagnostic)) (bool, *diagnostic.Diagnostic) {
	if err := validateRoot(root); err != nil {
		return false, err
	}
	initial, err := planner(root)
	if err != nil {
		return false, err
	}
	if check {
		return initial.Changed, nil
	}
	if e := os.MkdirAll(root, 0777); e != nil {
		return false, ioFailure()
	}
	absolute, e := filepath.Abs(root)
	if e != nil {
		return false, ioFailure()
	}
	canonical, e := filepath.EvalSymlinks(absolute)
	if e != nil {
		return false, ioFailure()
	}
	root = canonical
	lockPath := filepath.Join(root, lockName)
	meta, e := os.Lstat(lockPath)
	if e == nil && !meta.Mode().IsRegular() {
		return false, fail("E_OUTPUT_PATH", "Output lock must be a regular file.")
	}
	lock, e := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0666)
	if e != nil {
		return false, ioFailure()
	}
	defer lock.Close()
	if e := tryLock(lock); e != nil {
		return false, fail("E_OUTPUT_BUSY", "Another generator owns this output directory.")
	}
	defer unlock(lock)
	locked, err := planner(root)
	if err != nil {
		return false, err
	}
	if !locked.Changed {
		return false, nil
	}
	if err := commit(root, locked, nil); err != nil {
		return false, err
	}
	return true, nil
}
