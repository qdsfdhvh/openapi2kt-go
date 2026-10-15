package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigurationAdoptionNativeGuards(t *testing.T) {
	for _, kind := range []string{"suffix", "modified", "missing", "unsupported", "generation", "warning"} {
		t.Run(kind, func(t *testing.T) {
			dir, cfg, root := legacyConfigRoot(t)
			wanted := "E_MANIFEST_OWNER_MISMATCH"
			exit := 4
			extra := []string{}
			switch kind {
			case "suffix":
				extra = []string{"--model-name-suffix", "Dto"}
			case "modified":
				if e := os.WriteFile(filepath.Join(root, "org/example/config/models/Alpha.kt"), []byte("modified"), 0600); e != nil {
					t.Fatal(e)
				}
				wanted = "E_OUTPUT_MODIFIED"
			case "missing":
				if e := os.Remove(filepath.Join(root, "org/example/config/models/Alpha.kt")); e != nil {
					t.Fatal(e)
				}
				wanted = "E_OUTPUT_MISSING"
			case "unsupported":
				if e := os.WriteFile(filepath.Join(root, ".openapi-kt-manifest.json"), []byte(`{"version":3,"files":{}}`), 0600); e != nil {
					t.Fatal(e)
				}
				wanted = "E_MANIFEST_VERSION"
			case "generation":
				if e := os.WriteFile(filepath.Join(dir, "Alpha.json"), []byte("{bad"), 0600); e != nil {
					t.Fatal(e)
				}
				wanted = "E_JSON_INVALID"
				exit = 1
			case "warning":
				data, e := os.ReadFile(filepath.Join(repo, "tests/fixtures/cli/warning-policy.json"))
				if e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(dir, "Alpha.json"), data, 0600); e != nil {
					t.Fatal(e)
				}
				extra = []string{"--deny-warnings"}
				wanted = "E_WARNING_DENIED"
				exit = 1
			}
			before := snapshot(t, root)
			for _, check := range []bool{false, true} {
				args := append([]string{"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--diagnostic-format=json"}, extra...)
				if check {
					args = append(args, "--check")
				}
				r := result(t, invoke(t, repo, args...), exit)
				if len(r.Files) != 0 || len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != wanted {
					t.Fatal(r)
				}
				if kind == "warning" && len(r.Warnings) == 0 {
					t.Fatal("suppressed denied warnings")
				}
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("failed migration changed legacy output")
				}
			}
		})
	}
}
func TestConfigurationAdoptionHelp(t *testing.T) {
	for _, command := range []string{"generate", "validate", ""} {
		args := []string{"--help"}
		if command != "" {
			args = []string{command, "--help"}
		}
		out := invoke(t, repo, args...)
		if out.Code != 0 || len(out.Stderr) != 0 || bytes.Contains(out.Stdout, []byte("--adopt-legacy")) != (command == "generate") {
			t.Fatal(command, out)
		}
	}
}
func TestConfigurationAdoptionRequiresExplicitIntent(t *testing.T) {
	_, cfg, _ := legacyConfigRoot(t)
	for _, args := range [][]string{{"generate", "--config", cfg, "--adopt-legacy"}, {"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy=true"}, {"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--adopt-legacy"}, {"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--reconcile"}, {"validate", "--config", cfg, "--spec", "alpha", "--adopt-legacy"}, {"download", "--config", cfg, "--adopt-legacy"}, {"generate", "--input", "unused", "--output", "unused", "--package-name", "org.example.unused", "--adopt-legacy"}} {
		fails(t, invoke(t, repo, append(args, "--diagnostic-format=json")...), 2, "E_USAGE")
	}
}
