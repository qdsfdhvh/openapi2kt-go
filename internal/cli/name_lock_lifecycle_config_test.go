package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func lifecycleConfig(t testing.TB, base string, names []string, locked, multiRoot, sharedRuntime bool) string {
	t.Helper()
	text := "config_version=1\n[defaults]\noutput_dir='generated'\nmodel_name_suffix='Dto'\nvalidation=true\nhoist_shared_shapes=true\nqualify_hoisted_names=true\ndeduplicate_models=true\nlock_hoisted_names=" + strconv.FormatBool(locked) + "\n"
	if sharedRuntime {
		text += "runtime_package='org.example.lockruntime'\n"
	}
	for _, name := range names {
		text += "[[specs]]\nname=" + strconv.Quote(name) + "\ninput=" + strconv.Quote(name+".json") + "\npackage_name=" + strconv.Quote("org.example.lock"+name) + "\n"
		if multiRoot {
			text += "output_dir=" + strconv.Quote(name+"-output") + "\n"
		}
	}
	cfg := filepath.Join(base, "lifecycle.toml")
	put(t, cfg, []byte(text))
	return cfg
}

func lifecycleConfigInputs(t testing.TB, base string, names []string) {
	t.Helper()
	for _, name := range names {
		put(t, filepath.Join(base, name+".json"), read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
	}
}

func TestNameLockLifecycleNativeConfigSelectionRetainsPeersAndSharedRuntime(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		t.Run(action, func(t *testing.T) {
			base := t.TempDir()
			names := []string{"alpha", "beta"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, true, false, true)
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, base, args...), 0)
			root := filepath.Join(base, "generated")
			oldState := read(t, filepath.Join(root, namedLockPath("alpha")))
			before := snapshot(t, root)
			var manifest struct {
				Version   int
				Files     map[string]string
				Owners    map[string][]string
				NameLocks map[string]string `json:"name_locks"`
			}
			if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil {
				t.Fatal(err)
			}
			shared := "org/example/lockruntime/Patch.kt"
			for _, owner := range names {
				found := false
				for _, path := range manifest.Owners[owner] {
					found = found || path == shared
				}
				if !found {
					t.Fatal("fixture does not exercise shared runtime ownership", owner)
				}
			}
			// Selected lifecycle actions must not read an unselected spec input,
			// or require that peer to have the action's target lock bool.
			if err := os.Remove(filepath.Join(base, "beta.json")); err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(base, "alpha.json")
			lockGrow(t, input)
			selected := append(append([]string(nil), args...), "--spec=alpha")
			if action == "disable" {
				selected = append(selected, "--lock-hoisted-names=false")
			}
			selected = lifecycleAction(selected, action)
			checkOut := invoke(t, base, append(selected, "--check")...)
			if bytes.Contains(checkOut.Stdout, []byte(base)) {
				t.Fatal("selected lifecycle report leaked an absolute path")
			}
			checked := lifecycleMigrations(t, checkOut, 3)
			// Configured validation also accepts output/suffix with disable.
			validation := append(lifecycleValidate(selected), "--output", root, "--model-name-suffix=Dto")
			validated := lifecycleMigrations(t, invoke(t, base, validation...), 0)
			if len(checked) != 1 || !reflect.DeepEqual(checked, validated) || checked[0].Spec != "alpha" || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("selected lifecycle report/readonly behavior", checked, validated)
			}
			fresh := lifecycleCore(t, input, "org.example.lockalpha", "org.example.lockruntime", action == "reset")
			probe := lifecycleCore(t, input, "org.example.lockalpha", "org.example.lockruntime", true)
			alphaBefore := map[string]snap{}
			for _, path := range manifest.Owners["alpha"] {
				stored, exists := before[filepath.FromSlash(path)]
				if !exists {
					t.Fatal("selected membership is absent from the pre-action snapshot", path)
				}
				alphaBefore[path] = stored
			}
			lifecycleReport(t, checked[0], action, "alpha", oldState, probe.NameLock, alphaBefore, fresh.Files)
			committed := lifecycleMigrations(t, invoke(t, t.TempDir(), selected...), 0)
			if !reflect.DeepEqual(checked, committed) {
				t.Fatal("selected migration report changed across cwd/commit")
			}
			after := snapshot(t, root)
			for _, path := range append(manifest.Owners["beta"], namedLockPath("beta")) {
				key := filepath.FromSlash(path)
				old, hadOld := before[key]
				retained, exists := after[key]
				if !hadOld || !exists || !reflect.DeepEqual(old, retained) {
					t.Fatal("selected action changed retained peer/shared runtime", path)
				}
			}
			manifest.NameLocks = nil
			if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil || manifest.Version != 4 {
				t.Fatal("named lifecycle must retain v4", err, manifest)
			}
			if action == "disable" {
				if _, exists := manifest.NameLocks["alpha"]; exists || len(manifest.NameLocks) != 1 {
					t.Fatal("disable removed incorrect owner metadata", manifest.NameLocks)
				}
				if _, err := os.Stat(filepath.Join(root, namedLockPath("alpha"))); !os.IsNotExist(err) {
					t.Fatal("selected disabled metadata survived", err)
				}
			} else if !bytes.Equal(probe.NameLock, read(t, filepath.Join(root, namedLockPath("alpha")))) {
				t.Fatal("selected reset state differs from fresh core")
			}
		})
	}
}

func TestNameLockLifecycleNativeConfigFullMultiRootAndLastDisableV4(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		t.Run(action, func(t *testing.T) {
			base := t.TempDir()
			names := []string{"alpha", "beta"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, true, true, false)
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, base, args...), 0)
			old := map[string]map[string]snap{}
			states := map[string][]byte{}
			for _, name := range names {
				root := filepath.Join(base, name+"-output")
				old[name] = snapshot(t, root)
				states[name] = read(t, filepath.Join(root, namedLockPath(name)))
				lockGrow(t, filepath.Join(base, name+".json"))
			}
			if action == "disable" {
				lifecycleConfig(t, base, names, false, true, false)
				fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_DISABLED")
			}
			actionArgs := lifecycleAction(args, action)
			checkOut := invoke(t, base, append(actionArgs, "--check")...)
			if bytes.Contains(checkOut.Stdout, []byte(base)) {
				t.Fatal("multi-root lifecycle report leaked an absolute path")
			}
			checked := lifecycleMigrations(t, checkOut, 3)
			validated := lifecycleMigrations(t, invoke(t, base, lifecycleValidate(actionArgs)...), 0)
			if len(checked) != 2 || !reflect.DeepEqual(checked, validated) {
				t.Fatal("full action must apply to every selected owner", checked, validated)
			}
			for i, name := range names {
				root := filepath.Join(base, name+"-output")
				if !reflect.DeepEqual(old[name], snapshot(t, root)) {
					t.Fatal("full action check/validate changed root", name)
				}
				input := filepath.Join(base, name+".json")
				fresh := lifecycleCore(t, input, "org.example.lock"+name, "", action == "reset")
				probe := lifecycleCore(t, input, "org.example.lock"+name, "", true)
				lifecycleReport(t, checked[i], action, name, states[name], probe.NameLock, old[name], fresh.Files)
			}
			committed := lifecycleMigrations(t, invoke(t, base, actionArgs...), 0)
			if !reflect.DeepEqual(checked, committed) {
				t.Fatal("full action migration report differs on commit")
			}
			for _, name := range names {
				root := filepath.Join(base, name+"-output")
				fresh := lifecycleCore(t, filepath.Join(base, name+".json"), "org.example.lock"+name, "", action == "reset")
				lifecycleSources(t, root, fresh.Files)
				var manifest struct {
					Version   int
					NameLocks map[string]string `json:"name_locks"`
				}
				if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil || manifest.Version != 4 {
					t.Fatal("last named disable must preserve version4", err, manifest)
				}
				if action == "disable" && len(manifest.NameLocks) != 0 {
					t.Fatal("full disable left locked owners", manifest)
				}
			}
		})
	}
}

func TestNameLockLifecycleNativeConfigEffectiveMismatchAndActionIsNotAKey(t *testing.T) {
	base := t.TempDir()
	cfg := filepath.Join(base, "config.toml")
	root := filepath.Join(base, "generated")
	for _, command := range []string{"generate", "validate"} {
		for _, test := range []struct{ action, defaults, specs string }{
			{"reset", "lock_hoisted_names=false", ""},
			{"reset", "lock_hoisted_names=true\nhoist_shared_shapes=false", ""},
			{"disable", "lock_hoisted_names=true", ""},
			{"disable", "lock_hoisted_names=false\nhoist_shared_shapes=false", ""},
			{"disable", "lock_hoisted_names=false", "lock_hoisted_names=true"},
		} {
			put(t, cfg, []byte("config_version=1\n[defaults]\ninput='missing.json'\noutput_dir='generated'\npackage_name='org.example.lifecycle'\nhoist_shared_shapes=true\n"+strings.ReplaceAll(test.defaults, "hoist_shared_shapes=false", "")+"\n[[specs]]\nname='alpha'\n"+test.specs+"\n"))
			if strings.Contains(test.defaults, "hoist_shared_shapes=false") {
				put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "hoist_shared_shapes=true", "hoist_shared_shapes=false", 1)))
			}
			// An invalid state/input cannot mask an effective action mismatch.
			if err := os.MkdirAll(root, 0700); err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(root, ".openapi-kt-manifest.json"), []byte("synthetic invalid manifest"))
			before := snapshot(t, root)
			fails(t, invoke(t, base, command, "--config", cfg, "--name-lock-action="+test.action, "--diagnostic-format=json"), 1, "E_OPTIONS")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("effective mismatch performed output IO")
			}
		}
		// A global action is not a generation-option override, but every selected
		// owner still needs the matching effective bool.
		put(t, cfg, []byte("config_version=1\n[defaults]\ninput='missing.json'\noutput_dir='generated'\npackage_name='org.example.lifecycle'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='alpha'\n[[specs]]\nname='beta'\nlock_hoisted_names=false\n"))
		fails(t, invoke(t, base, command, "--config", cfg, "--name-lock-action=reset", "--diagnostic-format=json"), 1, "E_OPTIONS")
		fails(t, invoke(t, base, command, "--config", cfg, "--name-lock-action=disable", "--diagnostic-format=json"), 1, "E_OPTIONS")
	}
	for _, section := range []string{"[defaults]", "[[specs]]\nname='alpha'"} {
		put(t, cfg, []byte("config_version=1\n"+section+"\nname_lock_action='reset'\n"))
		fails(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 1, "E_CONFIG_UNKNOWN_FIELD")
	}
	put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\n"))
	fails(t, invoke(t, base, "generate", "--config", cfg, "--name-lock-action=reset", "--diagnostic-format=json"), 1, "E_CONFIG_EMPTY")
}

func TestNameLockLifecycleNativeConfigLaterRootSafetyAndRetainedState(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		base := t.TempDir()
		names := []string{"alpha", "beta"}
		lifecycleConfigInputs(t, base, names)
		cfg := lifecycleConfig(t, base, names, true, true, false)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		ar, br := filepath.Join(base, "alpha-output"), filepath.Join(base, "beta-output")
		lockGrow(t, filepath.Join(base, "alpha.json"))
		lockGrow(t, filepath.Join(base, "beta.json"))
		put(t, filepath.Join(br, "org/example/lockbeta/models/AlphaResponseDto.kt"), []byte("synthetic modified owned peer"))
		if action == "disable" {
			lifecycleConfig(t, base, names, false, true, false)
		}
		beforeA, beforeB := snapshot(t, ar), snapshot(t, br)
		actionArgs := lifecycleAction(args, action)
		for _, run := range [][]string{actionArgs, append(append([]string(nil), actionArgs...), "--check"), lifecycleValidate(actionArgs)} {
			fails(t, invoke(t, base, run...), 4, "E_OUTPUT_MODIFIED")
			if !reflect.DeepEqual(beforeA, snapshot(t, ar)) || !reflect.DeepEqual(beforeB, snapshot(t, br)) {
				t.Fatal("later lifecycle preflight failure changed an earlier root")
			}
		}
		// Selecting only alpha excludes beta's unsafe root from output preflight.
		lifecycleMigrations(t, invoke(t, base, append(actionArgs, "--spec=alpha")...), 0)
		if !reflect.DeepEqual(beforeB, snapshot(t, br)) {
			t.Fatal("partial action accessed the unselected root")
		}
	}
}

func TestNameLockLifecycleNativeConfigReconcileAndRetainedLockSafety(t *testing.T) {
	base := t.TempDir()
	names := []string{"alpha", "beta"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, true, false, false)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	root := filepath.Join(base, "generated")
	// Full action reconciles owners outside the active inventory, preserving
	// similarly named unowned metadata. Selected alpha alone still needs state.
	foreign := filepath.Join(root, namedLockPath("foreign"))
	put(t, foreign, []byte("synthetic unowned metadata"))
	betaState := filepath.Join(root, namedLockPath("beta"))
	original := read(t, betaState)
	put(t, betaState, []byte("synthetic modified retained/reconciled metadata"))
	lifecycleConfig(t, base, []string{"alpha"}, true, false, false)
	before := snapshot(t, root)
	for _, run := range [][]string{
		lifecycleAction(args, "reset"),
		append(lifecycleAction(args, "reset"), "--check"),
		append(lifecycleAction(args, "reset"), "--spec=alpha"),
		lifecycleValidate(lifecycleAction(args, "reset")),
		append(lifecycleValidate(lifecycleAction(args, "reset")), "--spec=alpha"),
	} {
		fails(t, invoke(t, base, run...), 4, "E_NAME_LOCK_MODIFIED")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("lifecycle erased malformed retained/reconciled metadata")
		}
	}
	put(t, betaState, original)
	lifecycleMigrations(t, invoke(t, base, lifecycleAction(args, "reset")...), 0)
	if _, err := os.Stat(betaState); !os.IsNotExist(err) {
		t.Fatal("full lifecycle did not reconcile removed owner metadata", err)
	}
	if !reflect.DeepEqual(before[namedLockPath("foreign")], snapshot(t, root)[namedLockPath("foreign")]) {
		t.Fatal("full lifecycle removed unowned metadata")
	}
}

func TestNameLockLifecycleNativeConfigRequiresSelectedOwnedState(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		for _, existing := range []bool{false, true} {
			base := t.TempDir()
			names := []string{"alpha", "beta"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, false, false, false)
			// Only beta starts locked. A v4 peer state never authorizes an
			// action against the selected, previously unlocked alpha owner.
			put(t, cfg, append(read(t, cfg), []byte("lock_hoisted_names=true\n")...))
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			root := filepath.Join(base, "generated")
			var before map[string]snap
			if existing {
				result(t, invoke(t, base, args...), 0)
				before = snapshot(t, root)
			}
			selected := append(lifecycleAction(args, action), "--spec=alpha")
			if action == "reset" {
				selected = append(selected, "--lock-hoisted-names=true")
			}
			for _, run := range [][]string{selected, append(append([]string(nil), selected...), "--check"), lifecycleValidate(selected)} {
				fails(t, invoke(t, base, run...), 4, "E_NAME_LOCK_LIFECYCLE")
				if existing && !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("missing selected state action changed peer/unlocked output")
				}
				if !existing {
					if _, err := os.Stat(root); !os.IsNotExist(err) {
						t.Fatal("configured lifecycle created a fresh output root", err)
					}
				}
			}
		}
	}
}
