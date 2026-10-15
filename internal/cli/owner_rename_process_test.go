package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func renameConfig(t testing.TB, cfg, from, to string) {
	t.Helper()
	text := string(read(t, cfg))
	text = strings.ReplaceAll(text, "name=\""+from+"\"", "name=\""+to+"\"")
	put(t, cfg, []byte(text))
}

func TestOwnerRenameNativePartialSharedAndSourceParity(t *testing.T) {
	for _, locked := range []bool{false, true} {
		base := t.TempDir()
		lifecycleConfigInputs(t, base, []string{"alpha", "beta"})
		cfg := lifecycleConfig(t, base, []string{"alpha", "beta"}, locked, false, true)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		root := filepath.Join(base, "generated")
		before := snapshot(t, root)
		renameConfig(t, cfg, "alpha", "gamma")
		if e := os.Remove(filepath.Join(base, "beta.json")); e != nil {
			t.Fatal(e)
		}
		selected := append(append([]string(nil), args...), "--spec=gamma", "--rename-owner=alpha:gamma")
		checked := invoke(t, base, append(selected, "--check")...)
		result(t, checked, 3)
		validated := invoke(t, base, lifecycleValidate(selected)...)
		result(t, validated, 0)
		var raw struct {
			OwnerRenames []struct {
				Old      string `json:"old_owner"`
				New      string `json:"new_owner"`
				Version  int
				Files    []struct{ Path, SHA256 string }
				NameLock json.RawMessage `json:"name_lock"`
			} `json:"owner_renames"`
		}
		if e := json.Unmarshal(checked.Stdout, &raw); e != nil || len(raw.OwnerRenames) != 1 {
			t.Fatal("missing migration report", e, string(checked.Stdout))
		}
		m := raw.OwnerRenames[0]
		if m.Old != "alpha" || m.New != "gamma" || len(m.Files) == 0 || (m.Version == 4) != locked || bytes.Contains(checked.Stdout, []byte(base)) {
			t.Fatal("incomplete report", string(checked.Stdout))
		}
		var manifest struct {
			Owners map[string][]string
			Files  map[string]string
		}
		if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); e != nil {
			t.Fatal(e)
		}
		if len(m.Files) != len(manifest.Owners["alpha"]) {
			t.Fatal("report membership differs")
		}
		for i, file := range m.Files {
			if file.Path != manifest.Owners["alpha"][i] || file.SHA256 != manifest.Files[file.Path] {
				t.Fatal("report file hashes differ", file)
			}
		}
		if locked {
			var lock struct {
				SHA256  string
				Symbols []lifecycleInventory
				Profile struct {
					Package, Suffix string
					Qualified       bool
				}
			}
			if e := json.Unmarshal(m.NameLock, &lock); e != nil {
				t.Fatal(e)
			}
			old := before[namedLockPath("alpha")].Bytes
			if lock.SHA256 != *lifecycleHash(old) || lock.Profile.Package != "org.example.lockalpha" || lock.Profile.Suffix != "Dto" || !lock.Profile.Qualified || !reflect.DeepEqual(lock.Symbols, lifecycleInventories(t, []byte(old))) {
				t.Fatal("report changed public names/descriptors/occurrences/profile", lock)
			}
		} else if len(m.NameLock) != 0 {
			t.Fatal("unlocked report includes lock")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("check/validate wrote output")
		}
		committed := invoke(t, t.TempDir(), selected...)
		result(t, committed, 0)
		var checkRaw, validRaw, commitRaw map[string]json.RawMessage
		for i, out := range []invocation{checked, validated, committed} {
			targets := []*map[string]json.RawMessage{&checkRaw, &validRaw, &commitRaw}
			if e := json.Unmarshal(out.Stdout, targets[i]); e != nil {
				t.Fatal(e)
			}
		}
		if !bytes.Equal(checkRaw["owner_renames"], validRaw["owner_renames"]) || !bytes.Equal(checkRaw["owner_renames"], commitRaw["owner_renames"]) {
			t.Fatal("readonly/apply report differs")
		}
		after := snapshot(t, root)
		for path, old := range before {
			if strings.HasSuffix(path, ".kt") || path == namedLockPath("beta") {
				if !reflect.DeepEqual(old, after[path]) {
					t.Fatal("Kotlin/peer rewritten", path)
				}
			}
		}
		if locked && before[namedLockPath("alpha")].Bytes != after[namedLockPath("gamma")].Bytes {
			t.Fatal("allocations changed")
		}
		// Subsequent generation must reuse the renamed state, and be a source/mtime noop.
		result(t, invoke(t, base, append(args, "--spec=gamma", "--check")...), 0)
		result(t, invoke(t, base, append(args, "--spec=gamma")...), 0)
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("ordinary post-rename generation changed bytes/mtime")
		}
		identity := append(append([]string(nil), args...), "--spec=gamma", "--rename-owner=gamma:gamma")
		result(t, invoke(t, base, append(identity, "--check")...), 0)
		result(t, invoke(t, base, identity...), 0)
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("identity rename changed output")
		}
		// Full rename does not reconcile an owner absent from the current config.
		text := string(read(t, cfg))
		cut := strings.Index(text, "[[specs]]\nname=\"beta\"")
		if cut < 0 {
			t.Fatal("missing synthetic peer config")
		}
		put(t, cfg, []byte(text[:cut]))
		result(t, invoke(t, base, append(args, "--rename-owner=gamma:gamma")...), 0)
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("full rename reconciled absent peer")
		}
	}
}

func TestOwnerRenameNativeFullMultiRootAndPartialIsolation(t *testing.T) {
	base := t.TempDir()
	names := []string{"alpha", "beta"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, true, true, false)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	renameConfig(t, cfg, "alpha", "gamma")
	renameConfig(t, cfg, "beta", "delta")
	full := append(append([]string(nil), args...), "--rename-owner=alpha:gamma", "--rename-owner=beta:delta")
	beforeA, beforeB := snapshot(t, filepath.Join(base, "alpha-output")), snapshot(t, filepath.Join(base, "beta-output"))
	result(t, invoke(t, base, append(full, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(full)...), 0)
	if !reflect.DeepEqual(beforeA, snapshot(t, filepath.Join(base, "alpha-output"))) || !reflect.DeepEqual(beforeB, snapshot(t, filepath.Join(base, "beta-output"))) {
		t.Fatal("full check wrote")
	}
	if e := os.Mkdir(filepath.Join(base, "beta-output", ".openapi-kt-stage-legacy"), 0700); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(base, "beta-output", ".openapi-kt-stage-legacy", "transaction.json"), []byte("old journal"))
	beforeB = snapshot(t, filepath.Join(base, "beta-output"))
	fails(t, invoke(t, base, full...), 4, "E_RECOVERY_REQUIRED")
	if !reflect.DeepEqual(beforeA, snapshot(t, filepath.Join(base, "alpha-output"))) {
		t.Fatal("later root failure changed first root")
	}
	result(t, invoke(t, base, append(args, "--spec=gamma", "--rename-owner=alpha:gamma")...), 0)
	if !reflect.DeepEqual(beforeB, snapshot(t, filepath.Join(base, "beta-output"))) {
		t.Fatal("partial discovered peer root")
	}
}

func TestOwnerRenameNativeGuardsAndHelp(t *testing.T) {
	base := t.TempDir()
	lifecycleConfigInputs(t, base, []string{"alpha", "beta"})
	cfg := lifecycleConfig(t, base, []string{"alpha", "beta"}, true, false, false)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	root := filepath.Join(base, "generated")
	before := snapshot(t, root)
	for _, extra := range [][]string{
		{"--rename-owner=alpha"}, {"--rename-owner=:alpha"}, {"--rename-owner=alpha:bad:name"},
		{"--rename-owner=alpha:gamma", "--rename-owner=alpha:gamma"},
		{"--spec=alpha", "--rename-owner=alpha:alpha", "--adopt-legacy"},
		{"--rename-owner=alpha:alpha", "--reconcile"},
		{"--rename-owner=alpha:alpha", "--name-lock-action=reset"},
	} {
		fails(t, invoke(t, base, append(args, extra...)...), 2, "E_USAGE")
	}
	for _, extra := range [][]string{
		{"--rename-owner=alpha:alpha"}, // Full selection must map beta too.
		{"--spec=alpha", "--rename-owner=alpha:beta"},
		{"--spec=alpha", "--rename-owner=absent:alpha"},
		{"--spec=beta", "--rename-owner=alpha:beta"},
	} {
		fails(t, invoke(t, base, append(args, extra...)...), 4, "E_OWNER_RENAME_INVALID")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("guard lost old output")
	}
	help := invoke(t, base, "generate", "--help")
	if help.Code != 0 || len(help.Stderr) != 0 {
		t.Fatal(help)
	}
	if !bytes.Contains(help.Stdout, []byte("--rename-owner <OLD:NEW>")) {
		t.Fatal("help missing rename")
	}
	renameConfig(t, cfg, "alpha", "gamma")
	selected := append(args, "--spec=gamma", "--rename-owner=alpha:gamma")
	fails(t, invoke(t, base, append(selected, "--package-name=org.example.other")...), 1, "E_NAME_LOCK_PROFILE")
	lockGrow(t, filepath.Join(base, "alpha.json"))
	for _, run := range [][]string{selected, append(selected, "--check"), lifecycleValidate(selected)} {
		fails(t, invoke(t, base, run...), 4, "E_OWNER_RENAME_CHANGED")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("profile/candidate drift lost output")
	}
}
