package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLockedOwnerAdoptionNativeWarningPolicyPreservesSource(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "input.json")
	root := filepath.Join(base, "generated")
	put(t, input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"empty":{"type":"object"}}}}}}`))
	result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.warning", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
	cfg := filepath.Join(base, "adopt.toml")
	put(t, cfg, []byte("config_version=1\ndefaults={output_dir='generated',package_name='org.example.warning',hoist_shared_shapes=true,lock_hoisted_names=true}\nspecs=[{name='catalog',input='input.json'}]\n"))
	before := snapshot(t, root)
	run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--deny-warnings", "--diagnostic-format=json"}
	for _, args := range [][]string{run, append(append([]string(nil), run...), "--check"), lifecycleValidate(run)} {
		out := result(t, invoke(t, base, args...), 1)
		if len(out.Diagnostics) != 1 || out.Diagnostics[0].Code != "E_WARNING_DENIED" || len(out.Warnings) != 1 {
			t.Fatal(out)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("warning denial changed source")
		}
	}
	result(t, invoke(t, base, append(run, "--allow-warning=W_JSON_ELEMENT_APPROXIMATION")...), 0)
}
