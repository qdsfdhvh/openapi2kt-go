package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigurationCanonicalSourceParityAndReconcile(t *testing.T) {
	dir, cfg := configFiles(t)
	var a, b map[string]any
	_ = json.Unmarshal([]byte(configDocument("Alpha")), &a)
	_ = json.Unmarshal([]byte(configDocument("Beta")), &b)
	a["components"].(map[string]any)["schemas"].(map[string]any)["Beta"] = b["components"].(map[string]any)["schemas"].(map[string]any)["Beta"]
	data, _ := json.Marshal(a)
	full := filepath.Join(dir, "Full.json")
	if e := os.WriteFile(full, data, 0600); e != nil {
		t.Fatal(e)
	}
	control := filepath.Join(dir, "control")
	legacy := result(t, invoke(t, repo, "generate", "--input", full, "--output", control, "--package-name", "org.example.config", "--validation", "--diagnostic-format=json"), 0)
	actual := result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(legacy.Files, actual.Files) {
		t.Fatal(legacy, actual)
	}
	root := filepath.Join(dir, "generated")
	for _, p := range actual.Files {
		want, e := os.ReadFile(filepath.Join(control, p))
		if e != nil {
			t.Fatal(e)
		}
		got, e := os.ReadFile(filepath.Join(root, p))
		if e != nil || !bytes.Equal(want, got) {
			t.Fatal("config source differs from whole canonical", p, e)
		}
	}
	raw, _ := os.ReadFile(cfg)
	cut := bytes.LastIndex(raw, []byte("[[specs]]"))
	if cut < 0 {
		t.Fatal("test fixture")
	}
	if e := os.WriteFile(cfg, raw[:cut], 0600); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("partial removed beta")
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 3)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("full check wrote")
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	var m struct{ Owners map[string][]string }
	mb, _ := os.ReadFile(filepath.Join(root, ".openapi-kt-manifest.json"))
	if e := json.Unmarshal(mb, &m); e != nil || len(m.Owners) != 1 || m.Owners["beta"] != nil {
		t.Fatal(m, e)
	}
}
func TestConfigurationWarningDenialAndLegacyMigration(t *testing.T) {
	dir, cfg := configFiles(t)
	root := filepath.Join(dir, "generated")
	result(t, invoke(t, repo, "generate", "--input", filepath.Join(dir, "Alpha.json"), "--output", root, "--package-name", "org.example.config", "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 4, "E_MANIFEST_OWNER_MIGRATION")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("config adopted legacy")
	}
	warning, _ := os.ReadFile(filepath.Join(repo, "tests/fixtures/cli/warning-policy.json"))
	if e := os.WriteFile(filepath.Join(dir, "Alpha.json"), warning, 0600); e != nil {
		t.Fatal(e)
	}
	out := invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--deny-warnings", "--diagnostic-format=json")
	r := result(t, out, 1)
	if len(r.Files) != 0 || len(r.Warnings) == 0 || len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != "E_WARNING_DENIED" {
		t.Fatal(r)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("config deny modified output")
	}
}
