package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfigurationMultiRootPreflightAndCheckNoIO(t *testing.T) {
	dir, cfg := multiRootConfig(t)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 3)
	a, b := filepath.Join(dir, "generated"), filepath.Join(dir, "generated-beta")
	for _, root := range []string{a, b} {
		if _, e := os.Stat(root); !os.IsNotExist(e) {
			t.Fatal("check created root", e)
		}
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	alphaFile := filepath.Join(dir, "Alpha.json")
	if e := os.WriteFile(alphaFile, []byte(strings.Replace(configDocument("Alpha"), `"type":"integer"`, `"type":"string"`, 1)), 0600); e != nil {
		t.Fatal(e)
	}
	putPath := filepath.Join(b, "org/example/config/models/Beta.kt")
	if e := os.WriteFile(putPath, []byte("modified owned file"), 0600); e != nil {
		t.Fatal(e)
	}
	beforeA, beforeB := snapshot(t, a), snapshot(t, b)
	for _, extra := range [][]string{nil, {"--check"}} {
		args := append([]string{"generate", "--config", cfg, "--diagnostic-format=json"}, extra...)
		fails(t, invoke(t, repo, args...), 4, "E_OUTPUT_MODIFIED")
	}
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("peer preflight error changed first root")
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("partial accessed/modified unselected root")
	}
}
func TestConfigurationMultiRootCanonicalSourceParity(t *testing.T) {
	dir, cfg := multiRootConfig(t)
	actual := result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	count := 0
	for _, spec := range []struct{ name, root string }{{"Alpha", "generated"}, {"Beta", "generated-beta"}} {
		control := filepath.Join(dir, "control-"+spec.name)
		r := result(t, invoke(t, repo, "generate", "--input", filepath.Join(dir, spec.name+".json"), "--output", control, "--package-name", "org.example.config", "--validation", "--diagnostic-format=json"), 0)
		count += len(r.Files)
		for _, path := range r.Files {
			expected, e := os.ReadFile(filepath.Join(control, path))
			if e != nil {
				t.Fatal(e)
			}
			got, e := os.ReadFile(filepath.Join(dir, spec.root, path))
			if e != nil || !bytes.Equal(expected, got) {
				t.Fatal(spec, path, e)
			}
		}
	}
	if len(actual.Files) != count {
		t.Fatal("different roots must retain separate relative-path report entries", actual)
	}
}
