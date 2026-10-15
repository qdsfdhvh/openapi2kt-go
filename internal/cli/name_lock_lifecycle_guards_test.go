package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func lifecycleRewriteState(t testing.TB, root string, state []byte) {
	t.Helper()
	var manifest map[string]json.RawMessage
	path := filepath.Join(root, ".openapi-kt-manifest.json")
	if err := json.Unmarshal(read(t, path), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["name_lock"], _ = json.Marshal(lifecycleHash(string(state)))
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, data)
	put(t, filepath.Join(root, ".openapi-kt-names.json"), state)
}

func TestNameLockLifecycleNativeBadStateOutputAndRecoveryPreserved(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		for _, kind := range []string{"missing", "modified", "malformed", "version", "manifest", "modified-owned", "unowned", "recovery"} {
			t.Run(action+"/"+kind, func(t *testing.T) {
				base := t.TempDir()
				input := lockInput(t, base)
				root := filepath.Join(base, "generated")
				result(t, invoke(t, base, lifecycleArgs(input, root, true)...), 0)
				state := filepath.Join(root, ".openapi-kt-names.json")
				code := ""
				switch kind {
				case "missing":
					if err := os.Remove(state); err != nil {
						t.Fatal(err)
					}
					code = "E_NAME_LOCK_MISSING"
				case "modified":
					put(t, state, append(read(t, state), ' '))
					code = "E_NAME_LOCK_MODIFIED"
				case "malformed":
					lifecycleRewriteState(t, root, []byte("{synthetic malformed owned state"))
					code = "E_NAME_LOCK_INVALID"
				case "version":
					lifecycleRewriteState(t, root, []byte(strings.Replace(string(read(t, state)), `"version":1`, `"version":2`, 1)))
					code = "E_NAME_LOCK_VERSION"
				case "manifest":
					put(t, filepath.Join(root, ".openapi-kt-manifest.json"), []byte(`{"version":3,"files":{},"name_lock":"invalid"}`))
					code = "E_MANIFEST_INVALID"
				case "modified-owned":
					put(t, filepath.Join(root, "org/example/dedup/models/AlphaResponseDto.kt"), []byte("synthetic manually edited owned model"))
					code = "E_OUTPUT_MODIFIED"
				case "unowned":
					lockGrow(t, input)
					put(t, filepath.Join(root, "org/example/dedup/models/AardvarkResponseDto.kt"), []byte("synthetic handwritten collision"))
					code = "E_OUTPUT_COLLISION"
				case "recovery":
					stage := filepath.Join(root, ".openapi-kt-stage-interrupted")
					if err := os.Mkdir(stage, 0700); err != nil {
						t.Fatal(err)
					}
					put(t, filepath.Join(stage, "transaction.json"), []byte("malformed retained journal; do not interpret"))
					put(t, filepath.Join(stage, "old-0"), read(t, state))
					code = "E_RECOVERY_REQUIRED"
				}
				before := snapshot(t, root)
				args := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
				for _, run := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
					out := invoke(t, base, run...)
					fails(t, out, 4, code)
					lifecycleNoMigrations(t, out)
					if !reflect.DeepEqual(before, snapshot(t, root)) {
						t.Fatal("lifecycle refusal changed output or recovery evidence")
					}
				}
			})
		}
	}
}

func TestNameLockLifecycleNativeUnownedMetadataIsNeverAdopted(t *testing.T) {
	base := t.TempDir()
	input := lockInput(t, base)
	for _, action := range []string{"reset", "disable"} {
		root := filepath.Join(base, action)
		result(t, invoke(t, base, lifecycleArgs(input, root, false)...), 0)
		state := lifecycleCore(t, input, "org.example.dedup", "", true).NameLock
		put(t, filepath.Join(root, ".openapi-kt-names.json"), state)
		before := snapshot(t, root)
		args := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
		for _, run := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
			fails(t, invoke(t, base, run...), 4, "E_OUTPUT_COLLISION")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("lifecycle adopted or removed unowned metadata")
			}
		}
	}
}

func TestNameLockLifecycleNativeShapeChangeAndRemovedOccurrences(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		for _, kind := range []string{"changed", "removed"} {
			t.Run(action+"/"+kind, func(t *testing.T) {
				base := t.TempDir()
				input := lockInput(t, base)
				root := filepath.Join(base, "generated")
				args := lifecycleArgs(input, root, true)
				result(t, invoke(t, base, args...), 0)
				old := read(t, filepath.Join(root, ".openapi-kt-names.json"))
				if kind == "changed" {
					// Change every occurrence of the shared request tree without
					// changing the historical source-pointer provenance.
					put(t, input, []byte(strings.ReplaceAll(string(read(t, input)), `"minLength": 2`, `"minLength": 4`)))
				} else {
					var doc, paths map[string]json.RawMessage
					if err := json.Unmarshal(read(t, input), &doc); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(doc["paths"], &paths); err != nil {
						t.Fatal(err)
					}
					delete(paths, "/beta")
					doc["paths"], _ = json.Marshal(paths)
					data, _ := json.Marshal(doc)
					put(t, input, data)
				}
				before := snapshot(t, root)
				code := "E_NAME_LOCK_CHANGED"
				if kind == "removed" {
					code = "E_NAME_LOCK_REMOVED"
				}
				fails(t, invoke(t, base, args...), 1, code)
				actionArgs := lifecycleAction(lifecycleArgs(input, root, action == "reset"), action)
				freshLocked := lifecycleCore(t, input, "org.example.dedup", "", true)
				fresh := lifecycleCore(t, input, "org.example.dedup", "", action == "reset")
				migrations := lifecycleMigrations(t, invoke(t, base, lifecycleValidate(actionArgs)...), 0)
				if len(migrations) != 1 {
					t.Fatal("direct migration must have one report", migrations)
				}
				lifecycleReport(t, migrations[0], action, "", old, freshLocked.NameLock, before, fresh.Files)
				observed := false
				for _, change := range migrations[0].Changes {
					observed = observed || kind == "changed" && change.ShapeChanged || kind == "removed" && change.Before != nil && change.After == nil
				}
				if !observed || !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("drift migration did not report shape/removal or readonly validation wrote")
				}
				lifecycleMigrations(t, invoke(t, base, actionArgs...), 0)
				lifecycleSources(t, root, fresh.Files)
			})
		}
	}
}

func TestNameLockLifecycleNativeWarningDenialPreservesOutput(t *testing.T) {
	for _, configured := range []bool{false, true} {
		for _, action := range []string{"reset", "disable"} {
			p := warningProject(t)
			args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.catalog", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"}
			if configured {
				cfg := filepath.Join(p.Root, "warning-lifecycle.toml")
				put(t, cfg, []byte("config_version=1\n[defaults]\ninput='catalog.json'\noutput_dir='generated'\npackage_name='org.example.catalog'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='alpha'\n"))
				args = []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			}
			result(t, invoke(t, repo, args...), 0)
			before := snapshot(t, p.Output)
			if action == "disable" {
				if configured {
					args = append(args, "--lock-hoisted-names=false")
				} else {
					for i, a := range args {
						if a == "--lock-hoisted-names" {
							args[i] = "--lock-hoisted-names=false"
						}
					}
				}
			}
			args = append(lifecycleAction(args, action), "--deny-warnings")
			for _, run := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
				out := invoke(t, repo, run...)
				r := result(t, out, 1)
				if len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != "E_WARNING_DENIED" || len(r.Warnings) == 0 {
					t.Fatal("lifecycle bypassed warning policy", r)
				}
				lifecycleNoMigrations(t, out)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("warning denial changed lifecycle output")
				}
			}
		}
	}
}
