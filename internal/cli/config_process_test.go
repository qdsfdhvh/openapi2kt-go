package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func configDocument(name string) string {
	return `{"openapi":"3.1.0","info":{"title":"Config Example","version":"1"},"paths":{},"components":{"schemas":{"` + name + `":{"type":"object","properties":{"id":{"type":"integer"}}}}}}`
}
func configFiles(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"Alpha", "Beta"} {
		if e := os.WriteFile(filepath.Join(dir, name+".json"), []byte(configDocument(name)), 0600); e != nil {
			t.Fatal(e)
		}
	}
	text := `config_version = 1
[defaults]
output_dir = 'generated'
package_name = 'org.example.config'
validation = true
[[specs]]
name = 'alpha'
input = 'Alpha.json'
[[specs]]
name = 'beta'
input = 'Beta.json'
`
	file := filepath.Join(dir, "config.toml")
	if e := os.WriteFile(file, []byte(text), 0600); e != nil {
		t.Fatal(e)
	}
	return dir, file
}
func TestConfigurationCanonicalControl(t *testing.T) {
	dir, _ := configFiles(t)
	var a, b map[string]any
	_ = json.Unmarshal([]byte(configDocument("Alpha")), &a)
	_ = json.Unmarshal([]byte(configDocument("Beta")), &b)
	a["components"].(map[string]any)["schemas"].(map[string]any)["Beta"] = b["components"].(map[string]any)["schemas"].(map[string]any)["Beta"]
	data, _ := json.Marshal(a)
	full := filepath.Join(dir, "Full.json")
	if e := os.WriteFile(full, data, 0600); e != nil {
		t.Fatal(e)
	}
	r := result(t, invoke(t, repo, "generate", "--input", full, "--output", filepath.Join(dir, "control"), "--package-name", "org.example.config", "--validation", "--diagnostic-format=json"), 0)
	if len(r.Files) == 0 {
		t.Fatal(r)
	}
}
func TestConfigurationNativeSelection(t *testing.T) {
	dir, cfg := configFiles(t)
	r := result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	if len(r.Files) < 2 {
		t.Fatal(r)
	}
	root := filepath.Join(dir, "generated")
	before := snapshot(t, root)
	result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	if e := os.Remove(filepath.Join(dir, "Beta.json")); e != nil {
		t.Fatal(e)
	}
	result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("partial lost unselected spec")
	}
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 4, "E_INPUT_IO")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("failed full generation changed output")
	}
}
