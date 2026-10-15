package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func multiRootConfig(t *testing.T) (string, string) {
	t.Helper()
	dir, cfg := configFiles(t)
	data, e := os.ReadFile(cfg)
	if e != nil {
		t.Fatal(e)
	}
	data = append(data, []byte("output_dir = 'generated-beta'\n")...)
	if e := os.WriteFile(cfg, data, 0600); e != nil {
		t.Fatal(e)
	}
	return dir, cfg
}
func TestConfigurationMultiRootNative(t *testing.T) {
	dir, cfg := multiRootConfig(t)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	a, b := filepath.Join(dir, "generated"), filepath.Join(dir, "generated-beta")
	beforeA, beforeB := snapshot(t, a), snapshot(t, b)
	result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("multi root noop/partial changed files")
	}
	if e := os.WriteFile(filepath.Join(dir, "Beta.json"), []byte("{bad"), 0600); e != nil {
		t.Fatal(e)
	}
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 1, "E_JSON_INVALID")
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("second spec failure changed either root")
	}
}
