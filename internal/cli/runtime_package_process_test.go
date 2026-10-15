package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func runtimeConfiguration(t *testing.T, shared bool) (string, string) {
	t.Helper()
	dir := t.TempDir()
	input := filepath.Join(repo, "tests/fixtures/http/shared-runtime.json")
	pkg := ""
	if shared {
		pkg = "runtime_package = 'org.example.shared.runtime'\n"
	}
	cfg := filepath.Join(dir, "openapi-kt.toml")
	data := "config_version=1\n[defaults]\ninput='" + filepath.ToSlash(input) + "'\noutput_dir='generated'\nmodel_name_suffix='Dto'\nvalidation=true\n" + pkg + "[[specs]]\nname='alpha'\npackage_name='org.example.runtimealpha'\n[[specs]]\nname='beta'\npackage_name='org.example.runtimebeta'\n"
	if e := os.WriteFile(cfg, []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
	return cfg, filepath.Join(dir, "generated")
}
func TestRuntimePackageDefaultControl(t *testing.T) {
	cfg, root := runtimeConfiguration(t, false)
	r := result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	for _, check := range [][]string{nil, {"--check"}} {
		result(t, invoke(t, repo, append([]string{"generate", "--config", cfg, "--diagnostic-format=json"}, check...)...), 0)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("default runtime noop")
	}
	for _, pkg := range []string{"runtimealpha", "runtimebeta"} {
		for _, helper := range []string{"Patch.kt", "LongAsStringSerializer.kt"} {
			if _, e := os.Stat(filepath.Join(root, "org/example", pkg, "models", helper)); e != nil {
				t.Fatal(r, e)
			}
		}
	}
}
func TestRuntimePackageSharedNative(t *testing.T) {
	cfg, root := runtimeConfiguration(t, true)
	r := result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	for _, helper := range []string{"Patch.kt", "LongAsStringSerializer.kt"} {
		data, e := os.ReadFile(filepath.Join(root, "org/example/shared/runtime", helper))
		if e != nil || !bytes.Contains(data, []byte("package org.example.shared.runtime")) {
			t.Fatal(r, e)
		}
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("shared runtime partial/check changed output")
	}
}
