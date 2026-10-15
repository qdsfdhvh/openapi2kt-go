package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"openapi-kt/core"
)

type lifecycleInventory struct {
	Role        string   `json:"role"`
	Shape       string   `json:"shape"`
	Classifier  string   `json:"classifier"`
	Descriptor  string   `json:"descriptor"`
	Occurrences []string `json:"occurrences"`
}

type lifecycleChange struct {
	Role             string  `json:"role"`
	Occurrence       string  `json:"occurrence"`
	Before           *string `json:"before,omitempty"`
	After            *string `json:"after,omitempty"`
	DescriptorBefore *string `json:"descriptor_before,omitempty"`
	DescriptorAfter  *string `json:"descriptor_after,omitempty"`
	ShapeChanged     bool    `json:"shape_changed"`
}

type lifecycleFile struct {
	Path   string  `json:"path"`
	Before *string `json:"before,omitempty"`
	After  *string `json:"after,omitempty"`
}

type lifecycleMigration struct {
	Action  string               `json:"action"`
	Spec    string               `json:"spec,omitempty"`
	Before  []lifecycleInventory `json:"before"`
	After   []lifecycleInventory `json:"after"`
	Changes []lifecycleChange    `json:"changes"`
	Files   []lifecycleFile      `json:"files"`
	Impact  string               `json:"impact"`
}

func lifecycleArgs(input, root string, locked bool) []string {
	args := append(lockArgs(input, root), "--qualify-hoisted-names")
	if !locked {
		for i, a := range args {
			if a == "--lock-hoisted-names" {
				args[i] = "--lock-hoisted-names=false"
			}
		}
	}
	return args
}

func lifecycleAction(args []string, action string) []string {
	return append(append([]string(nil), args...), "--name-lock-action="+action)
}

func lifecycleValidate(args []string) []string {
	args = append([]string(nil), args...)
	args[0] = "validate"
	return args
}

func lifecycleCore(t testing.TB, input, pkg, runtime string, locked bool) core.GenerationArtifacts {
	t.Helper()
	a, err := core.Generate(core.GenerationRequest{Document: read(t, input), Format: core.JSON, Options: core.GenerationOptions{
		PackageName: pkg, RuntimePackage: runtime, ModelNameSuffix: "Dto", Validation: true,
		HoistSharedShapes: true, QualifyHoistedNames: true, DeduplicateModels: true, LockHoistedNames: locked,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func lifecycleInventories(t testing.TB, data []byte) []lifecycleInventory {
	t.Helper()
	var state struct {
		Profile struct{ Package string }
		Entries []struct {
			Role, Shape, Name string
			Occurrences       []string
		}
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	out := make([]lifecycleInventory, 0, len(state.Entries))
	for _, entry := range state.Entries {
		classifier := state.Profile.Package + ".models." + entry.Name
		out = append(out, lifecycleInventory{entry.Role, entry.Shape, classifier, classifier, entry.Occurrences})
	}
	return out
}

func lifecycleMigrations(t testing.TB, out invocation, code int) []lifecycleMigration {
	t.Helper()
	result(t, out, code)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out.Stdout, &raw); err != nil {
		t.Fatal(err)
	}
	var migrations []lifecycleMigration
	if err := json.Unmarshal(raw["name_lock_migrations"], &migrations); err != nil || len(migrations) == 0 {
		t.Fatal("missing additive name_lock_migrations", err, string(out.Stdout))
	}
	for _, migration := range migrations {
		if migration.Before == nil || migration.After == nil || migration.Changes == nil || migration.Files == nil || migration.Impact == "" {
			t.Fatal("incomplete migration report", string(out.Stdout))
		}
		for _, file := range migration.Files {
			if filepath.IsAbs(file.Path) || strings.Contains(file.Path, "\\") || strings.HasPrefix(file.Path, "../") || !strings.HasSuffix(file.Path, ".kt") {
				t.Fatal("migration file path is not relative Kotlin", file)
			}
			for _, hash := range []*string{file.Before, file.After} {
				if hash != nil {
					decoded, err := hex.DecodeString(*hash)
					if err != nil || len(decoded) != sha256.Size || strings.ToLower(*hash) != *hash {
						t.Fatal("migration file must report lowercase SHA256", file)
					}
				}
			}
		}
	}
	return migrations
}

func lifecycleNoMigrations(t testing.TB, out invocation) {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out.Stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["name_lock_migrations"]; exists {
		t.Fatal("ordinary/error report gained migration metadata", string(out.Stdout))
	}
}

func lifecycleSources(t testing.TB, root string, files []core.GeneratedFile) {
	t.Helper()
	actual := map[string]string{}
	for path, file := range snapshot(t, root) {
		if strings.HasSuffix(path, ".kt") {
			actual[filepath.ToSlash(path)] = file.Bytes
		}
	}
	want := map[string]string{}
	for _, file := range files {
		want[file.Path] = file.Content
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatal("native lifecycle source set differs from fresh core output")
	}
}

func lifecycleHash(data string) *string {
	hash := sha256.Sum256([]byte(data))
	value := hex.EncodeToString(hash[:])
	return &value
}

func lifecycleFileChanges(old map[string]snap, files []core.GeneratedFile) []lifecycleFile {
	before, after := map[string]string{}, map[string]string{}
	for path, file := range old {
		if strings.HasSuffix(path, ".kt") {
			before[filepath.ToSlash(path)] = file.Bytes
		}
	}
	for _, file := range files {
		after[file.Path] = file.Content
	}
	paths := map[string]bool{}
	for path := range before {
		paths[path] = true
	}
	for path := range after {
		paths[path] = true
	}
	out := []lifecycleFile{}
	for path := range paths {
		b, hadBefore := before[path]
		a, hasAfter := after[path]
		if hadBefore && hasAfter && a == b {
			continue
		}
		file := lifecycleFile{Path: path}
		if hadBefore {
			file.Before = lifecycleHash(b)
		}
		if hasAfter {
			file.After = lifecycleHash(a)
		}
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func lifecycleReport(t testing.TB, migration lifecycleMigration, action, spec string, oldState, freshState []byte, old map[string]snap, files []core.GeneratedFile) {
	t.Helper()
	lifecycleReportInventories(t, migration, action, spec, lifecycleInventories(t, oldState), lifecycleInventories(t, freshState), old, files)
}

func lifecycleReportInventories(t testing.TB, migration lifecycleMigration, action, spec string, beforeInventory, afterInventory []lifecycleInventory, old map[string]snap, files []core.GeneratedFile) {
	t.Helper()
	if migration.Action != action || migration.Spec != spec || !reflect.DeepEqual(migration.Before, beforeInventory) || !reflect.DeepEqual(migration.After, afterInventory) {
		t.Fatal("migration action/spec/inventories differ from persisted and fresh states", migration)
	}
	if !reflect.DeepEqual(migration.Files, lifecycleFileChanges(old, files)) {
		t.Fatal("migration changed-file inventory differs from actual source SHA256 changes", migration.Files, lifecycleFileChanges(old, files))
	}
	before, after := map[string]lifecycleInventory{}, map[string]lifecycleInventory{}
	for _, inventory := range migration.Before {
		for _, occurrence := range inventory.Occurrences {
			before[inventory.Role+":"+occurrence] = inventory
		}
	}
	for _, inventory := range migration.After {
		for _, occurrence := range inventory.Occurrences {
			after[inventory.Role+":"+occurrence] = inventory
		}
	}
	seen := map[string]bool{}
	for _, change := range migration.Changes {
		key := change.Role + ":" + change.Occurrence
		b, hadBefore := before[key]
		a, hasAfter := after[key]
		if seen[key] || !hadBefore && !hasAfter {
			t.Fatal("duplicate or unknown migration occurrence", change)
		}
		seen[key] = true
		if hadBefore != (change.Before != nil) || hasAfter != (change.After != nil) || hadBefore != (change.DescriptorBefore != nil) || hasAfter != (change.DescriptorAfter != nil) {
			t.Fatal("migration occurrence presence differs from inventories", change)
		}
		if hadBefore && (*change.Before != b.Classifier || *change.DescriptorBefore != b.Descriptor) || hasAfter && (*change.After != a.Classifier || *change.DescriptorAfter != a.Descriptor) {
			t.Fatal("migration occurrence classifier/descriptor differs from inventory", change)
		}
		if hadBefore && hasAfter && change.ShapeChanged != (b.Shape != a.Shape) {
			t.Fatal("migration occurrence shape_changed differs from inventories", change)
		}
	}
	for key, b := range before {
		a, exists := after[key]
		if (!exists || b.Shape != a.Shape || b.Classifier != a.Classifier || b.Descriptor != a.Descriptor) && !seen[key] {
			t.Fatal("migration omitted changed/removed occurrence", key)
		}
	}
	for key := range after {
		if _, exists := before[key]; !exists && !seen[key] {
			t.Fatal("migration omitted added occurrence", key)
		}
	}
}

func TestNameLockLifecycleNativeResetDisableFreshParityAndReports(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		t.Run(action, func(t *testing.T) {
			base := t.TempDir()
			input := lockInput(t, base)
			root := filepath.Join(base, "generated")
			args := lifecycleArgs(input, root, true)
			ordinary := invoke(t, base, args...)
			result(t, ordinary, 0)
			lifecycleNoMigrations(t, ordinary)
			state := read(t, filepath.Join(root, ".openapi-kt-names.json"))
			put(t, filepath.Join(root, "Notes.kt"), []byte("synthetic handwritten note"))
			// An earlier same-shape representative has a different fresh qualified
			// name. Reusing the stored lock retains Alpha; reset/disable select Aardvark.
			lockGrow(t, input)
			before := snapshot(t, root)
			freshLocked := lifecycleCore(t, input, "org.example.dedup", "", true)
			fresh := lifecycleCore(t, input, "org.example.dedup", "", action == "reset")
			if !reflect.DeepEqual(fresh.Files, freshLocked.Files) || action == "disable" && fresh.NameLock != nil {
				t.Fatal("fresh lock probe must retain actual unlocked source bytes")
			}
			actionArgs := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
			rootBefore, err := os.Stat(root)
			if err != nil {
				t.Fatal(err)
			}
			checkOut := invoke(t, base, append(actionArgs, "--check")...)
			if bytes.Contains(checkOut.Stdout, []byte(base)) {
				t.Fatal("migration report leaked an absolute source/output path")
			}
			checked := lifecycleMigrations(t, checkOut, 3)
			validated := lifecycleMigrations(t, invoke(t, base, lifecycleValidate(actionArgs)...), 0)
			rootAfter, err := os.Stat(root)
			if err != nil || !rootBefore.ModTime().Equal(rootAfter.ModTime()) {
				t.Fatal("lifecycle check/validate changed output directory", err)
			}
			if len(checked) != 1 || !reflect.DeepEqual(checked, validated) || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("lifecycle check/validate report differs or modified output")
			}
			deleteOwnedNote := before["Notes.kt"]
			delete(before, "Notes.kt") // Handwritten files never enter source migration inventory.
			lifecycleReport(t, checked[0], action, "", state, freshLocked.NameLock, before, fresh.Files)
			hasRename, hasRemoval, hasAddition := false, false, false
			for _, change := range checked[0].Changes {
				if change.Before != nil && change.After != nil && *change.Before != *change.After {
					hasRename = true
					if change.DescriptorBefore == nil || change.DescriptorAfter == nil || *change.DescriptorBefore == *change.DescriptorAfter {
						t.Fatal("real classifier rename must report descriptor rename", change)
					}
				}
			}
			for _, file := range checked[0].Files {
				hasRemoval = hasRemoval || file.After == nil
				hasAddition = hasAddition || file.Before == nil
			}
			if !hasRename || !hasRemoval || !hasAddition {
				t.Fatal("fixture did not exercise actual source/classifier/descriptor rename", checked[0])
			}
			committed := lifecycleMigrations(t, invoke(t, t.TempDir(), actionArgs...), 0)
			if !reflect.DeepEqual(checked, committed) {
				t.Fatal("migration report is not deterministic across readonly/commit/cwd")
			}
			after := snapshot(t, root)
			if !reflect.DeepEqual(deleteOwnedNote, after["Notes.kt"]) {
				t.Fatal("lifecycle changed handwritten file")
			}
			// Compare only manifest-owned Kotlin, preserving the unrelated note.
			if err := os.Remove(filepath.Join(root, "Notes.kt")); err != nil {
				t.Fatal(err)
			}
			lifecycleSources(t, root, fresh.Files)
			var manifest map[string]json.RawMessage
			if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil {
				t.Fatal(err)
			}
			if action == "reset" {
				if string(manifest["version"]) != "3" || !bytes.Equal(fresh.NameLock, read(t, filepath.Join(root, ".openapi-kt-names.json"))) {
					t.Fatal("reset did not commit canonical fresh state in v3")
				}
			} else {
				if string(manifest["version"]) != "1" || manifest["name_lock"] != nil {
					t.Fatal("disable did not remove direct metadata ownership")
				}
				if _, err := os.Stat(filepath.Join(root, ".openapi-kt-names.json")); !os.IsNotExist(err) {
					t.Fatal("disable left owned metadata", err)
				}
			}
			stable := snapshot(t, root)
			plain := lifecycleArgs(input, root, action == "reset")
			for _, tail := range [][]string{nil, {"--check"}} {
				out := invoke(t, base, append(plain, tail...)...)
				result(t, out, 0)
				lifecycleNoMigrations(t, out)
			}
			if !reflect.DeepEqual(stable, snapshot(t, root)) {
				t.Fatal("ordinary generation after migration is not a no-op")
			}
			if action == "reset" {
				// Both spellings parse, and an already fresh reset has no output
				// differences even though its explicit report remains available.
				again := append(append([]string(nil), plain...), "--name-lock-action", "reset")
				for _, run := range [][]string{again, append(append([]string(nil), again...), "--check"), lifecycleValidate(again)} {
					migrations := lifecycleMigrations(t, invoke(t, base, run...), 0)
					if len(migrations) != 1 || len(migrations[0].Changes) != 0 || len(migrations[0].Files) != 0 || !reflect.DeepEqual(migrations[0].Before, migrations[0].After) {
						t.Fatal("already fresh reset reported output changes", migrations)
					}
				}
			} else {
				fails(t, invoke(t, base, actionArgs...), 4, "E_NAME_LOCK_LIFECYCLE")
			}
			if !reflect.DeepEqual(stable, snapshot(t, root)) {
				t.Fatal("repeated lifecycle action changed stable output")
			}
		})
	}
}

func TestNameLockLifecycleNativeFlagsAndHelpPreIO(t *testing.T) {
	base := t.TempDir()
	input, root := filepath.Join(base, "missing.json"), filepath.Join(base, "absent")
	for _, command := range []string{"generate", "validate"} {
		t.Run(command, func(t *testing.T) {
			h := invoke(t, base, command, "--help")
			if h.Code != 0 || !strings.Contains(string(h.Stdout), "--name-lock-action") || !strings.Contains(string(h.Stdout), "reset") || !strings.Contains(string(h.Stdout), "disable") {
				t.Fatal("missing lifecycle help", h)
			}
			for _, tail := range [][]string{
				{"--name-lock-action"}, {"--name-lock-action="}, {"--name-lock-action=RESET"}, {"--name-lock-action=unknown"},
				{"--name-lock-action=reset", "--name-lock-action=reset"}, {"--name-lock-action=reset", "--name-lock-action=disable"},
				{"--name-lock-action=disable"},
			} {
				args := lifecycleArgs(input, root, true)
				args[0] = command
				fails(t, invoke(t, base, append(args, tail...)...), 2, "E_USAGE")
			}
			args := lifecycleArgs(input, root, false)
			args[0] = command
			fails(t, invoke(t, base, lifecycleAction(args, "reset")...), 2, "E_USAGE")
		})
	}
	cfg := filepath.Join(base, "config.toml")
	put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='absent'\n"))
	for _, tail := range [][]string{{"--adopt-legacy", "--spec=alpha"}, {"--reconcile"}} {
		args := append([]string{"generate", "--config", cfg, "--name-lock-action=reset", "--diagnostic-format=json"}, tail...)
		fails(t, invoke(t, base, args...), 2, "E_USAGE")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("bad lifecycle arguments touched output", err)
	}
}

func TestNameLockLifecycleNativeRequiresOwnedStateAndSameProfile(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		for _, existing := range []bool{false, true} {
			base := t.TempDir()
			input := lockInput(t, base)
			root := filepath.Join(base, "generated")
			if existing {
				result(t, invoke(t, base, lifecycleArgs(input, root, false)...), 0)
			}
			var before map[string]snap
			if existing {
				before = snapshot(t, root)
			}
			args := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
			for _, run := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
				out := invoke(t, base, run...)
				fails(t, out, 4, "E_NAME_LOCK_LIFECYCLE")
				lifecycleNoMigrations(t, out)
				if existing && !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("absent-owned-state refusal changed v1 output")
				}
				if !existing {
					if _, err := os.Stat(root); !os.IsNotExist(err) {
						t.Fatal("lifecycle created a fresh output root", err)
					}
				}
			}
		}
		base := t.TempDir()
		input := lockInput(t, base)
		root := filepath.Join(base, "generated")
		result(t, invoke(t, base, lifecycleArgs(input, root, true)...), 0)
		before := snapshot(t, root)
		for _, mismatch := range []string{"--package-name=org.example.other", "--model-name-suffix=Other", "--qualify-hoisted-names=false"} {
			args := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
			key := strings.SplitN(mismatch, "=", 2)[0]
			for i, a := range args {
				if strings.SplitN(a, "=", 2)[0] == key {
					args[i] = mismatch
				}
			}
			for _, run := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
				fails(t, invoke(t, base, run...), 1, "E_NAME_LOCK_PROFILE")
			}
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("lifecycle profile rejection mutated output")
		}
		// The one-shot flag is required; false alone still refuses stored state.
		fails(t, invoke(t, base, lifecycleArgs(input, root, false)...), 4, "E_MANIFEST_VERSION")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("ordinary disable bypassed lifecycle ownership")
		}
	}
}
