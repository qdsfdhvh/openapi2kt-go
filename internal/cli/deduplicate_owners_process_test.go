package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeduplicateConfigPartialOwnerAndSharedRuntime(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.toml")
	root := filepath.Join(dir, "out")
	input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
	text := fmt.Sprintf("config_version=1\n[defaults]\noutput_dir='out'\nvalidation=true\ndeduplicate_models=true\nmodel_name_suffix='Dto'\nruntime_package='org.example.shared.runtime'\n[[specs]]\nname='a'\ninput=%q\npackage_name='org.example.owneralpha'\n[[specs]]\nname='b'\ninput=%q\npackage_name='org.example.ownerbeta'\n", input, input)
	if e := os.WriteFile(cfg, []byte(text), 0600); e != nil {
		t.Fatal(e)
	}
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	before := snapshot(t, root)
	r := result(t, invoke(t, repo, append(args, "--deduplicate-models=false")...), 1)
	if r.Diagnostics[0].Code != "E_CONFIG_OVERRIDE" || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("multi override changed output", r)
	}
	selected := append(append([]string{}, args...), "--spec=a", "--deduplicate-models=false")
	result(t, invoke(t, repo, selected...), 0)
	after := snapshot(t, root)
	for path, state := range before {
		if strings.HasPrefix(path, "org/example/ownerbeta/") || strings.HasPrefix(path, "org/example/shared/runtime/") {
			if !reflect.DeepEqual(state, after[path]) {
				t.Fatal("unselected owner or shared runtime changed", path)
			}
		}
	}
	beta, _ := os.ReadFile(filepath.Join(root, "org/example/ownerbeta/api/models/BetaResponseDto.kt"))
	alpha, _ := os.ReadFile(filepath.Join(root, "org/example/owneralpha/api/models/BetaResponseDto.kt"))
	if !strings.Contains(string(beta), "public typealias") || !strings.Contains(string(alpha), "public data class") {
		t.Fatal("selected false lost or unselected rewritten")
	}
	result(t, invoke(t, repo, append(selected, "--check")...), 0)
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("partial check writes")
	}
}
