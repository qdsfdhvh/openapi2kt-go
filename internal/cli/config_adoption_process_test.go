package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func legacyConfigRoot(t *testing.T) (string, string, string) {
	t.Helper()
	dir, cfg := configFiles(t)
	root := filepath.Join(dir, "generated")
	result(t, invoke(t, repo, "generate", "--input", filepath.Join(dir, "Alpha.json"), "--output", root, "--package-name", "org.example.config", "--validation", "--diagnostic-format=json"), 0)
	return dir, cfg, root
}
func TestConfigurationAdoptionControl(t *testing.T) {
	_, cfg, root := legacyConfigRoot(t)
	before := snapshot(t, root)
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 4, "E_MANIFEST_OWNER_MIGRATION")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("implicit adoption")
	}
}
func TestConfigurationExplicitLegacyAdoption(t *testing.T) {
	dir, cfg, root := legacyConfigRoot(t)
	if e := os.Remove(filepath.Join(dir, "Beta.json")); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	args := []string{"generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--diagnostic-format=json"}
	result(t, invoke(t, repo, append(args, "--check")...), 3)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("adoption check changed legacy")
	}
	result(t, invoke(t, repo, args...), 0)
	after := snapshot(t, root)
	for path, old := range before {
		if path == ".openapi-kt-manifest.json" {
			continue
		}
		if !reflect.DeepEqual(old, after[path]) {
			t.Fatal("metadata migration changed Kotlin/lock", path)
		}
	}
	var manifest struct {
		Version int
		Owners  map[string][]string
	}
	data, e := os.ReadFile(filepath.Join(root, ".openapi-kt-manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(data, &manifest); e != nil || manifest.Version != 2 || len(manifest.Owners) != 1 || len(manifest.Owners["alpha"]) == 0 {
		t.Fatal(manifest, e)
	}
	for _, extra := range [][]string{nil, {"--check"}} {
		result(t, invoke(t, repo, append(args, extra...)...), 0)
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--check", "--diagnostic-format=json"), 0)
	fails(t, invoke(t, repo, "generate", "--input", filepath.Join(dir, "Alpha.json"), "--output", root, "--package-name", "org.example.config", "--validation", "--diagnostic-format=json"), 4, "E_MANIFEST_VERSION")
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("adopted rerun/legacy refusal changed output")
	}
}
