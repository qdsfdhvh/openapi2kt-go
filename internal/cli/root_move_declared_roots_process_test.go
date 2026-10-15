package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func rootMoveDeclaredConfig(t testing.TB, base string, names []string, defaultsOutput string, specOutputs map[string]string, input string) string {
	t.Helper()
	text := "config_version=1\n[defaults]\ninput=" + strconv.Quote(input) + "\npackage_name='org.example.binding'\nvalidation=true\n" + defaultsOutput
	for _, name := range names {
		text += "[[specs]]\nname=" + strconv.Quote(name) + "\n" + specOutputs[name]
	}
	cfg := filepath.Join(base, "root-move.toml")
	put(t, cfg, []byte(text))
	return cfg
}

func rootMoveInvocation(args []string, mode string) []string {
	run := append([]string(nil), args...)
	if mode == "check" {
		run = append(run, "--check")
	}
	if mode == "validate" {
		run = lifecycleValidate(run)
	}
	return run
}

func requireRootMoveInventoryRefusalBeforeIO(t testing.TB, out invocation) {
	t.Helper()
	fails(t, out, 4, "E_OUTPUT_ROOT_MOVE_INVALID")
	var r struct {
		Metrics struct {
			InputBytes   int64            `json:"input_bytes"`
			PlannedFiles int64            `json:"planned_files"`
			PhaseNS      map[string]int64 `json:"phase_ns"`
		} `json:"metrics"`
	}
	if e := json.Unmarshal(out.Stdout, &r); e != nil {
		t.Fatal(e)
	}
	if len(r.Metrics.PhaseNS) != 1 {
		t.Fatal("invalid declared inventory entered state/input/output phases", r.Metrics)
	}
	if _, ok := r.Metrics.PhaseNS["config"]; !ok || r.Metrics.InputBytes != 0 || r.Metrics.PlannedFiles != 0 {
		t.Fatal("root inventory must reject before state and input reads", r.Metrics)
	}
}

func TestRootMoveNativeRejectsOmittedAndEmptyDeclaredRoots(t *testing.T) {
	for _, names := range [][]string{{"alpha"}, {"alpha", "beta"}} {
		for _, kind := range []string{"omitted", "empty-default", "empty-spec", "empty-spec-overrides-default", "one-empty-peer"} {
			if len(names) == 1 && kind == "one-empty-peer" {
				continue
			}
			for _, mode := range []string{"check", "validate", "apply"} {
				t.Run(strconv.Itoa(len(names))+"/"+kind+"/"+mode, func(t *testing.T) {
					base := t.TempDir()
					put(t, filepath.Join(base, "input.json"), read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
					cfg := rootMoveDeclaredConfig(t, base, names, "output_dir='source'\n", nil, "input.json")
					ordinary := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
					result(t, invoke(t, base, ordinary...), 0)
					source := filepath.Join(base, "source")
					put(t, filepath.Join(source, "Notes.kt"), []byte("// handwritten source"))
					before := snapshot(t, source)
					rootBefore, e := os.Stat(source)
					if e != nil {
						t.Fatal(e)
					}
					defaults := ""
					specs := map[string]string{}
					switch kind {
					case "empty-default":
						defaults = "output_dir=''\n"
					case "empty-spec":
						for _, name := range names {
							specs[name] = "output_dir=\"\"\n"
						}
					case "empty-spec-overrides-default":
						defaults = "output_dir='source'\n"
						for _, name := range names {
							specs[name] = "output_dir=''\n"
						}
					case "one-empty-peer":
						defaults = "output_dir='source'\n"
						specs[names[1]] = "output_dir=''\n"
					}
					rootMoveDeclaredConfig(t, base, names, defaults, specs, "input.json")
					target := filepath.Join(base, "missing", "target")
					move := append(append([]string(nil), ordinary...), "--move-output-from", source, "--output", target, "--metrics")
					requireRootMoveInventoryRefusalBeforeIO(t, invoke(t, base, rootMoveInvocation(move, mode)...))
					if !reflect.DeepEqual(before, snapshot(t, source)) {
						t.Fatal("invalid inventory changed source paths/bytes/modes/mtimes or handwritten file")
					}
					rootAfter, e := os.Stat(source)
					if e != nil {
						t.Fatal(e)
					}
					if rootBefore.Mode() != rootAfter.Mode() || !rootBefore.ModTime().Equal(rootAfter.ModTime()) {
						t.Fatal("invalid inventory changed source root")
					}
					if _, e := os.Stat(filepath.Dir(target)); !os.IsNotExist(e) {
						t.Fatal("invalid inventory created target parent")
					}
				})
			}
		}
	}
}

func TestRootMoveNativeEmptyDeclaredRootRefusesBeforeMissingInputOrState(t *testing.T) {
	for _, kind := range []string{"missing-input", "malformed-input", "missing-state", "malformed-state"} {
		for _, mode := range []string{"check", "validate", "apply"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				base := t.TempDir()
				source := filepath.Join(base, "source")
				if e := os.Mkdir(source, 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(source, "Notes.kt"), []byte("// handwritten source"))
				input := "input.json"
				if kind == "missing-input" {
					input = "missing.json"
				} else {
					put(t, filepath.Join(base, input), []byte("{"))
				}
				if kind != "missing-state" {
					put(t, filepath.Join(source, ".openapi-kt-manifest.json"), []byte("malformed manifest must not be read"))
				}
				cfg := rootMoveDeclaredConfig(t, base, []string{"alpha"}, "output_dir=''\n", nil, input)
				before := snapshot(t, source)
				target := filepath.Join(base, "missing", "target")
				args := []string{"generate", "--config", cfg, "--move-output-from", source, "--output", target, "--metrics", "--diagnostic-format=json"}
				requireRootMoveInventoryRefusalBeforeIO(t, invoke(t, base, rootMoveInvocation(args, mode)...))
				if !reflect.DeepEqual(before, snapshot(t, source)) {
					t.Fatal("early inventory refusal changed source")
				}
				if _, e := os.Stat(filepath.Dir(target)); !os.IsNotExist(e) {
					t.Fatal("early inventory refusal created target parent")
				}
			})
		}
	}
}

func TestRootMoveNativeEmptyTOMLOutputStillAllowsOrdinaryOverride(t *testing.T) {
	base := t.TempDir()
	put(t, filepath.Join(base, "input.json"), read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
	cfg := rootMoveDeclaredConfig(t, base, []string{"alpha"}, "output_dir='source'\n", map[string]string{"alpha": "output_dir=''\n"}, "input.json")
	source := filepath.Join(base, "source")
	args := []string{"generate", "--config", cfg, "--output", source, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	before := snapshot(t, source)
	result(t, invoke(t, base, append(args, "--check")...), 0)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, source)) {
		t.Fatal("ordinary empty output override semantics changed")
	}
}
