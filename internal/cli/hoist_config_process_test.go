package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHoistConfigPresenceAndDependencyBeforeIO(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.toml")
	out := filepath.Join(dir, "out")
	input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
	text := fmt.Sprintf("config_version=1\n[defaults]\nvalidation=true\nhoist_shared_shapes=true\nqualify_hoisted_names=true\noutput_dir='out'\nmodel_name_suffix='Dto'\n[[specs]]\nname='a'\ninput=%q\npackage_name='org.example.hoistconfig'\n", input)
	put := func(s string) {
		if e := os.WriteFile(cfg, []byte(s), 0600); e != nil {
			t.Fatal(e)
		}
	}
	put(text)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	alias := filepath.Join(out, "org/example/hoistconfig/api/models/BetaResponseDto.kt")
	check := func(want string) {
		b, e := os.ReadFile(alias)
		if e != nil || !strings.Contains(string(b), want) {
			t.Fatal(want, e, string(b))
		}
	}
	check("org.example.hoistconfig.models.AlphaResponseDto")
	before := snapshot(t, out)
	result(t, invoke(t, repo, append(args, "--qualify-hoisted-names=false", "--check")...), 3)
	if !reflect.DeepEqual(before, snapshot(t, out)) {
		t.Fatal("changed check writes")
	}
	result(t, invoke(t, repo, append(args, "--qualify-hoisted-names=false")...), 0)
	check("org.example.hoistconfig.models.ResponseDto")
	disabled := append(append([]string{}, args...), "--hoist-shared-shapes=false", "--qualify-hoisted-names=false")
	result(t, invoke(t, repo, disabled...), 0)
	check("public data class")
	before = snapshot(t, out)
	result(t, invoke(t, repo, append(disabled, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, out)) {
		t.Fatal("noop writes")
	}
	// Defaults true/spec false is effective invalid, diagnosed before a missing input.
	put(strings.Replace(text, input, filepath.Join(dir, "missing.json"), 1) + "hoist_shared_shapes=false\n")
	r := result(t, invoke(t, repo, args...), 1)
	if r.Diagnostics[0].Code != "E_OPTIONS" || !reflect.DeepEqual(before, snapshot(t, out)) {
		t.Fatal("dependency didn't protect original output before IO", r)
	}
	put(strings.Replace(text, "hoist_shared_shapes=true", "hoist_shared_shapes='true'", 1))
	r = result(t, invoke(t, repo, args...), 1)
	if r.Diagnostics[0].Code != "E_CONFIG_INVALID" || !reflect.DeepEqual(before, snapshot(t, out)) {
		t.Fatal("invalid settings writes", r)
	}
	for _, flag := range []string{"--hoist-shared-shapes=", "--qualify-hoisted-names=bad", "--qualify-hoisted-names"} {
		result(t, invoke(t, repo, append(deduplicateArgs(t.TempDir()), flag)...), 2)
	}
	result(t, invoke(t, repo, append(deduplicateArgs(t.TempDir()), "--hoist-shared-shapes", "--hoist-shared-shapes=false")...), 2)
	result(t, invoke(t, repo, "download", "--config", cfg, "--hoist-shared-shapes", "--diagnostic-format=json"), 2)
}
