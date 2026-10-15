package output

import (
	"openapi-kt/internal/diagnostic"
	"path/filepath"
)

// Only the strict version-1 migration refusal admits the legacy decoder. Other
// manifest errors never become an adoption/reset path. The source version is
// retained so planning treats even an identical file set as metadata migration.
func readOwnerManifest(root, adopt string, generated map[string][]byte) (*manifest, *diagnostic.Diagnostic) {
	return readOwnerManifestKind(root, adopt, generated, false)
}
func readOwnerManifestKind(root, adopt string, generated map[string][]byte, nameLocked bool) (*manifest, *diagnostic.Diagnostic) {
	previous, err := readManifestKind(root, true, nameLocked)
	if err == nil || err.Code != "E_MANIFEST_OWNER_MIGRATION" || adopt == "" {
		return previous, err
	}
	legacy, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	if legacy == nil {
		return nil, fail("E_MANIFEST_OWNER_MIGRATION", "Legacy ownership changed while planning; retry with a consistent output tree.")
	}
	mismatch := func() (*manifest, *diagnostic.Diagnostic) {
		return nil, fail("E_MANIFEST_OWNER_MISMATCH", "Legacy adoption requires an identical generated path and content-hash set; preserve the original generation options.")
	}
	if len(legacy.Files) != len(generated) {
		return mismatch()
	}
	for _, path := range sortedKeys(legacy.Files) {
		expected, exists := generated[path]
		if !exists || digest(expected) != legacy.Files[path] {
			return mismatch()
		}
		if err := inspectParents(root, path); err != nil {
			return nil, err
		}
		actual, exists, err := readRegular(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fail("E_OUTPUT_MISSING", "Restore missing legacy generated files before ownership migration.")
		}
		if digest(actual) != legacy.Files[path] {
			return nil, fail("E_OUTPUT_MODIFIED", "A previously generated file was modified. Restore or relocate it before generation.")
		}
	}
	owners := map[string][]string{}
	if len(legacy.Files) != 0 {
		owners[adopt] = sortedKeys(legacy.Files)
	}
	legacy.Owners = &owners
	return legacy, nil
}
