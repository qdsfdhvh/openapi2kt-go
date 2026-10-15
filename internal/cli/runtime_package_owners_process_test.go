package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRuntimePackageHelpAndExplicitEmpty(t *testing.T) {
	for _, cmd := range []string{"generate", "validate"} {
		out := invoke(t, repo, cmd, "--help")
		if out.Code != 0 || len(out.Stderr) != 0 || !bytes.Contains(out.Stdout, []byte("--runtime-package")) {
			t.Fatal(cmd, out)
		}
	}
	cfg, root := runtimeConfiguration(t, true)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--runtime-package=", "--diagnostic-format=json"), 0)
	for _, h := range []string{"Patch.kt", "LongAsStringSerializer.kt"} {
		if _, e := os.Stat(filepath.Join(root, "org/example/runtimealpha/models", h)); e != nil {
			t.Fatal(e)
		}
		if _, e := os.Stat(filepath.Join(root, "org/example/shared/runtime", h)); !os.IsNotExist(e) {
			t.Fatal("empty override inherited runtime default")
		}
	}
}
func TestRuntimePackageOwnerLifecycle(t *testing.T) {
	cfg, root := runtimeConfiguration(t, true)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	shared := []string{"org/example/shared/runtime/Patch.kt", "org/example/shared/runtime/LongAsStringSerializer.kt"}
	var m struct{ Owners map[string][]string }
	load := func() {
		b, e := os.ReadFile(filepath.Join(root, ".openapi-kt-manifest.json"))
		if e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal(b, &m); e != nil {
			t.Fatal(e)
		}
	}
	load()
	for _, p := range shared {
		for _, owner := range []string{"alpha", "beta"} {
			found := false
			for _, path := range m.Owners[owner] {
				found = found || path == p
			}
			if !found {
				t.Fatal("lost shared owner", owner, p, m)
			}
		}
	}
	old := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--spec", "alpha", "--runtime-package", "org.example.shifted.runtime")...), 0)
	shifted := snapshot(t, root)
	for _, p := range shared {
		if !reflect.DeepEqual(old[p], shifted[p]) {
			t.Fatal("shift removed peer runtime", p)
		}
	}
	load()
	for _, p := range m.Owners["alpha"] {
		if strings.HasPrefix(p, "org/example/shared/runtime/") {
			t.Fatal("old alpha helper membership retained")
		}
	}
	raw, e := os.ReadFile(cfg)
	if e != nil {
		t.Fatal(e)
	}
	raw = bytes.Replace(raw, []byte("org.example.shared.runtime"), []byte("org.example.shifted.runtime"), 1)
	cut := bytes.LastIndex(raw, []byte("[[specs]]"))
	if cut < 0 {
		t.Fatal("fixture")
	}
	if e := os.WriteFile(cfg, raw[:cut], 0600); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--spec", "alpha")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("partial removed inactive beta")
	}
	result(t, invoke(t, repo, append(args, "--check")...), 3)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("reconcile check wrote")
	}
	result(t, invoke(t, repo, args...), 0)
	for _, p := range shared {
		if _, e := os.Stat(filepath.Join(root, p)); !os.IsNotExist(e) {
			t.Fatal("last owner failed to clean old helper", p, e)
		}
	}
	if e := os.WriteFile(cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\n"), 0600); e != nil {
		t.Fatal(e)
	}
	result(t, invoke(t, repo, append(args, "--reconcile")...), 0)
	for p := range snapshot(t, root) {
		if strings.HasSuffix(p, ".kt") {
			t.Fatal("empty reconcile left owned helper/model", p)
		}
	}
}
func TestRuntimePackageInventoryRootGuardAndOverrides(t *testing.T) {
	for _, kind := range []string{"explicit", "implicitPeer", "invalid", "multiOverride"} {
		t.Run(kind, func(t *testing.T) {
			cfg, root := runtimeConfiguration(t, true)
			result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
			b, e := os.ReadFile(cfg)
			if e != nil {
				t.Fatal(e)
			}
			code := "E_CONFIG_RUNTIME_ROOTS"
			extra := []string{"--spec", "alpha"}
			switch kind {
			case "explicit":
				b = append(b, []byte("output_dir='other'\n")...)
			case "implicitPeer":
				b = bytes.Replace(b, []byte("org.example.shared.runtime"), []byte("org.example.runtimebeta.models"), 1)
				b = append(b, []byte("runtime_package=''\noutput_dir='other'\n")...)
			case "invalid":
				extra = append(extra, "--runtime-package", "kotlin.unsafe")
				code = "E_PACKAGE_INVALID"
			case "multiOverride":
				extra = []string{"--runtime-package", "org.example.one.runtime"}
				code = "E_CONFIG_OVERRIDE"
			}
			if e := os.WriteFile(cfg, b, 0600); e != nil {
				t.Fatal(e)
			}
			before := snapshot(t, root)
			fails(t, invoke(t, repo, append([]string{"generate", "--config", cfg, "--diagnostic-format=json"}, extra...)...), 1, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("root preflight changed old output")
			}
			if _, e := os.Stat(filepath.Join(filepath.Dir(cfg), "other")); !os.IsNotExist(e) {
				t.Fatal("created peer root", e)
			}
		})
	}
}
