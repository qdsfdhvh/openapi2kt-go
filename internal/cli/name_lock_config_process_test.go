package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func namedLockPath(owner string) string {
	h := sha256.Sum256([]byte(owner))
	return ".openapi-kt-names-" + hex.EncodeToString(h[:]) + ".json"
}
func lockConfig(t testing.TB, base string, names []string) string {
	t.Helper()
	cfg := filepath.Join(base, "generator.toml")
	s := "config_version=1\n[defaults]\ninput=\"input.json\"\noutput_dir=\"generated\"\nmodel_name_suffix=\"Dto\"\nvalidation=true\nhoist_shared_shapes=true\nlock_hoisted_names=true\nruntime_package=\"org.example.lockruntime\"\n"
	for _, name := range names {
		s += "[[specs]]\nname=\"" + name + "\"\npackage_name=\"org.example.lock" + name + "\"\n"
		if name == "gamma" {
			s += "lock_hoisted_names=false\n"
		}
	}
	put(t, cfg, []byte(s))
	return cfg
}
func TestNameLockConfigNativeMixedOwnersStateLifecycle(t *testing.T) {
	base := t.TempDir()
	input := lockInput(t, base)
	cfg := lockConfig(t, base, []string{"alpha", "beta", "gamma"})
	root := filepath.Join(base, "generated")
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, append(args, "--check")...), 3)
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("freshconfigcheck createdroot")
	}
	v := append([]string(nil), args...)
	v[0] = "validate"
	result(t, invoke(t, base, v...), 0)
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("freshconfigvalidate wrote")
	}
	result(t, invoke(t, base, args...), 0)
	var m struct {
		Version   int
		NameLocks map[string]string `json:"name_locks"`
		Owners    map[string][]string
	}
	if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil || m.Version != 4 || len(m.NameLocks) != 2 {
		t.Fatal(e, m)
	}
	originals := map[string][]byte{}
	for _, owner := range []string{"alpha", "beta"} {
		originals[owner] = read(t, filepath.Join(root, namedLockPath(owner)))
	}
	before := snapshot(t, root)
	result(t, invoke(t, base, append(args, "--check")...), 0)
	result(t, invoke(t, base, v...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("configcheck/validate wrote")
	}
	// Single false override applies only to selected unlocked peer; it cannot
	// drop other owners' state or silently disable an already locked owner.
	result(t, invoke(t, base, append(args, "--spec", "gamma", "--lock-hoisted-names=false")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("false peer selection dropped locks")
	}
	fails(t, invoke(t, base, append(args, "--spec", "alpha", "--lock-hoisted-names=false")...), 4, "E_NAME_LOCK_DISABLED")
	fails(t, invoke(t, base, append(args, "--lock-hoisted-names=false")...), 1, "E_CONFIG_OVERRIDE")
	lockGrow(t, input)
	partial := append(args, "--spec", "alpha")
	result(t, invoke(t, base, partial...), 0)
	actual, e := core.Generate(core.GenerationRequest{Document: read(t, input), Format: core.Auto, PreviousNameLock: originals["alpha"], Options: core.GenerationOptions{PackageName: "org.example.lockalpha", RuntimePackage: "org.example.lockruntime", ModelNameSuffix: "Dto", Validation: true, HoistSharedShapes: true, LockHoistedNames: true}})
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range actual.Files {
		if !bytes.Equal(read(t, filepath.Join(root, f.Path)), []byte(f.Content)) {
			t.Fatal("configcorefilediffers", f.Path)
		}
	}
	if !bytes.Equal(actual.NameLock, read(t, filepath.Join(root, namedLockPath("alpha")))) {
		t.Fatal("configstate differscore")
	}
	after := snapshot(t, root)
	if !reflect.DeepEqual(before[namedLockPath("beta")], after[namedLockPath("beta")]) {
		t.Fatal("peer lock rewritten")
	}
	for _, p := range m.Owners["beta"] {
		if !reflect.DeepEqual(before[p], after[p]) {
			t.Fatal("unselected peer changed", p)
		}
	}
	// Full inventory reconcile removes beta's owned metadata, retains alpha,
	// and never downgrades v4 when the last locked owner is removed.
	cfg = lockConfig(t, base, []string{"alpha", "gamma"})
	result(t, invoke(t, base, args...), 0)
	if _, e := os.Stat(filepath.Join(root, namedLockPath("beta"))); !os.IsNotExist(e) {
		t.Fatal("beta state not removed")
	}
	cfg = lockConfig(t, base, []string{"gamma"})
	result(t, invoke(t, base, args...), 0)
	m.NameLocks = nil // Unmarshal into a nonnil map does not clear removed entries.
	if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil || m.Version != 4 || len(m.NameLocks) != 0 {
		t.Fatal("implicitversiondowngrade", e)
	}
}
func TestNameLockConfigInventoryStrictAndPreIO(t *testing.T) {
	base := t.TempDir()
	cfg := filepath.Join(base, "config.toml")
	root := filepath.Join(base, "generated")
	for _, c := range []struct{ settings, code string }{
		{`lock_hoisted_names="true"`, "E_CONFIG_INVALID"}, {`lock_hoisted_names=1`, "E_CONFIG_INVALID"}, {"hoist_shared_shapes=false\nlock_hoisted_names=true", "E_OPTIONS"},
	} {
		text := "config_version=1\n[defaults]\ninput=\"missing.json\"\noutput_dir=\"generated\"\npackage_name=\"org.example.lock\"\n" + c.settings + "\n[[specs]]\nname=\"alpha\"\n"
		put(t, cfg, []byte(text))
		fails(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 1, c.code)
		if _, e := os.Stat(root); !os.IsNotExist(e) {
			t.Fatal("preflight wrote")
		}
	}
	put(t, cfg, []byte("config_version=1\n[defaults]\ninput=\"missing.json\"\noutput_dir=\"generated\"\npackage_name=\"org.example.lock\"\n[[specs]]\nname=\"alpha\"\n[[specs]]\nname=\"beta\"\nlock_hoisted_names=true\n"))
	fails(t, invoke(t, base, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 1, "E_OPTIONS")
	// Removed download tables are rejected even during local generation,
	// before reading missing inputs or state.
	put(t, cfg, []byte("config_version=1\n[[specs]]\nname=\"alpha\"\ninput=\"missing.json\"\n[specs.download]\nurl=\"https://example.test/schema\"\nlock_hoisted_names=true\n"))
	fails(t, invoke(t, base, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 1, "E_CONFIG_UNKNOWN_FIELD")
	put(t, cfg, []byte("config_version=1\n[defaults]\ninput=\"missing.json\"\noutput_dir=\"generated\"\npackage_name=\"org.example.lock\"\n[[specs]]\nname=\"alpha\"\n"))
	fails(t, invoke(t, base, "validate", "--config", cfg, "--output", root, "--diagnostic-format=json"), 2, "E_USAGE")
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("invalidconfigurationIO")
	}
}
func TestNameLockConfigExactLegacyAdoption(t *testing.T) {
	base := t.TempDir()
	input := lockInput(t, base)
	root := filepath.Join(base, "generated")
	args := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.lockalpha", "--runtime-package=org.example.lockruntime", "--model-name-suffix=Dto", "--validation", "--hoist-shared-shapes", "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	before := snapshot(t, root)
	cfg := lockConfig(t, base, []string{"alpha"})
	selected := []string{"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(selected, "--check")...), 3)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("adoptcheck mutated")
	}
	result(t, invoke(t, base, selected...), 0)
	after := snapshot(t, root)
	for p, s := range before {
		if strings.HasSuffix(p, ".kt") && !reflect.DeepEqual(s, after[p]) {
			t.Fatal("metadataonlyadoption rewrote Kotlin")
		}
	}
	if len(read(t, filepath.Join(root, namedLockPath("alpha")))) == 0 {
		t.Fatal("noadoptedstate")
	}
}
