package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func configWarningFixture(t *testing.T) (string, string, string, []string) {
	t.Helper()
	base := t.TempDir()
	cfg := filepath.Join(base, "policy.toml")
	output := filepath.Join(base, "generated")
	put(t, filepath.Join(base, "warning.json"), read(t, filepath.Join(repo, "tests/fixtures/cli/warning-policy.json")))
	text := "config_version=1\n[defaults]\ninput='warning.json'\noutput_dir='generated'\n[[specs]]\nname='alpha'\npackage_name='org.example.policyalpha'\n[[specs]]\nname='beta'\npackage_name='org.example.policybeta'\n"
	put(t, cfg, []byte(text))
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	return base, cfg, output, args
}

func TestConfigWarningPolicyNativePerSpecAndOverrides(t *testing.T) {
	base, cfg, output, args := configWarningFixture(t)
	original := string(read(t, cfg))
	before := snapshot(t, output)
	text := strings.Replace(original, "[defaults]", "[defaults]\ndeny_warnings=true\nallow_warnings=['W_ALL_OF_APPROXIMATION','W_JSON_ELEMENT_APPROXIMATION']", 1)
	text = strings.Replace(text, "name='alpha'", "name='alpha'\nallow_warnings=[]", 1)
	text = strings.Replace(text, "name='beta'", "name='beta'\ndeny_warnings=false\nallow_warnings=[]", 1)
	put(t, cfg, []byte(text))
	for _, command := range [][]string{args, append(append([]string(nil), args...), "--check"), lifecycleValidate(args)} {
		rejected := warningDenied(t, invoke(t, base, command...))
		if len(rejected.Diagnostics) != 2 || len(rejected.Warnings) != 4 || !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("per-spec policy lost warnings or old output", rejected)
		}
	}
	text = strings.Replace(text, "name='alpha'\nallow_warnings=[]", "name='alpha'", 1)
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 0)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("allowed policy changed bytes/mtimes")
	}
	for _, flags := range [][]string{{"--deny-warnings"}, {"--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION"}} {
		rejected := warningDenied(t, invoke(t, base, append(append([]string(nil), args...), flags...)...))
		if len(rejected.Diagnostics) != 2 || len(rejected.Warnings) != 4 {
			t.Fatal("CLI policy did not override every generated spec", rejected)
		}
	}
	result(t, invoke(t, base, append(append([]string(nil), args...), "--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION", "--allow-warning=W_JSON_ELEMENT_APPROXIMATION", "--check")...), 0)
	// Ordinary partial generation retains its no-read contract for unselected inputs.
	betaInput := filepath.Join(base, "absent-beta.json")
	text = strings.Replace(text, "name='beta'", "name='beta'\ninput='absent-beta.json'", 1)
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, append(append([]string(nil), args...), "--spec=alpha", "--check")...), 0)
	if _, err := os.Stat(betaInput); !os.IsNotExist(err) {
		t.Fatal("created unselected input")
	}
	if !reflect.DeepEqual(before, snapshot(t, output)) {
		t.Fatal("readonly policy changed state")
	}
}

func TestConfigWarningPolicyInvalidBeforeInputAndOutput(t *testing.T) {
	for _, policy := range []string{"deny_warnings=false\nallow_warnings=['W_ALL_OF_APPROXIMATION']", "deny_warnings=true\nallow_warnings=['W_*']", "deny_warnings=true\nallow_warnings=['W_secret_VALUE']"} {
		base := t.TempDir()
		cfg := filepath.Join(base, "policy.toml")
		put(t, cfg, []byte("config_version=1\n[defaults]\ninput='missing.json'\noutput_dir='absent/out'\npackage_name='org.example.policy'\n"+policy+"\n[[specs]]\nname='alpha'\n"))
		for _, command := range []string{"generate", "validate"} {
			rejected := fails(t, invoke(t, base, command, "--config", cfg, "--diagnostic-format=json"), 1, "E_CONFIG_INVALID")
			if len(rejected.Warnings) != 0 {
				t.Fatal("read input before policy configuration")
			}
		}
		if _, err := os.Stat(filepath.Join(base, "absent")); !os.IsNotExist(err) {
			t.Fatal("invalid policy touched output")
		}
	}
}

func TestConfigWarningPolicyMigrationFailuresPreserveOutput(t *testing.T) {
	for _, action := range []string{"root-move", "owner-rename", "lock-reset"} {
		base := t.TempDir()
		names := []string{"alpha", "beta"}
		lifecycleConfigInputs(t, base, names)
		for _, name := range names {
			put(t, filepath.Join(base, name+".json"), read(t, filepath.Join(repo, "tests/fixtures/cli/warning-policy.json")))
		}
		cfg := lifecycleConfig(t, base, names, true, false, false)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		root := filepath.Join(base, "generated")
		before := snapshot(t, root)
		text := strings.Replace(string(read(t, cfg)), "[defaults]", "[defaults]\ndeny_warnings=true", 1)
		run := append([]string(nil), args...)
		switch action {
		case "root-move":
			run = append(run, "--move-output-from", root, "--output", filepath.Join(base, "moved"), "--spec=alpha")
		case "owner-rename":
			text = strings.Replace(text, "name=\"alpha\"", "name=\"gamma\"", 1)
			run = append(run, "--spec=gamma", "--rename-owner=alpha:gamma")
		case "lock-reset":
			run = append(run, "--spec=alpha", "--name-lock-action=reset")
		}
		put(t, cfg, []byte(text))
		for _, command := range [][]string{run, append(append([]string(nil), run...), "--check"), lifecycleValidate(run)} {
			denied := warningDenied(t, invoke(t, base, command...))
			expected := 2
			if action == "root-move" {
				expected = 4
			}
			if len(denied.Diagnostics) != expected || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("migration policy lost full warning scope or output", action, denied)
			}
		}
		text = strings.Replace(text, "deny_warnings=true", "deny_warnings=true\nallow_warnings=['W_ALL_OF_APPROXIMATION','W_JSON_ELEMENT_APPROXIMATION']", 1)
		put(t, cfg, []byte(text))
		expectedCheck := 3
		if action == "lock-reset" {
			expectedCheck = 0
		} // Empty allocation reset is a validated no-op.
		result(t, invoke(t, base, append(append([]string(nil), run...), "--check")...), expectedCheck)
		result(t, invoke(t, base, lifecycleValidate(run)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("migration check/validate wrote output", action)
		}
	}
}

func TestConfigWarningPolicyLockedAdoption(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "input.json")
	root := filepath.Join(base, "generated")
	put(t, input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"empty":{"type":"object"}}}}}}`))
	result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.policy", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
	cfg := filepath.Join(base, "adopt.toml")
	text := "config_version=1\ndefaults={output_dir='generated',package_name='org.example.policy',hoist_shared_shapes=true,lock_hoisted_names=true,deny_warnings=true}\nspecs=[{name='catalog',input='input.json'}]\n"
	put(t, cfg, []byte(text))
	before := snapshot(t, root)
	run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	for _, command := range [][]string{run, append(append([]string(nil), run...), "--check"), lifecycleValidate(run)} {
		denied := warningDenied(t, invoke(t, base, command...))
		if len(denied.Diagnostics) != 1 || !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("adoption ignored persisted warning policy", denied)
		}
	}
	put(t, cfg, []byte(strings.Replace(text, "deny_warnings=true", "deny_warnings=true,allow_warnings=['W_JSON_ELEMENT_APPROXIMATION']", 1)))
	result(t, invoke(t, base, run...), 0)
	result(t, invoke(t, base, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
}
