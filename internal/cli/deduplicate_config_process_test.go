package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeduplicateNativeConfigPresenceAndGuards(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
	cfg := filepath.Join(dir, "config.toml")
	root := filepath.Join(dir, "out")
	write := func(value string) {
		text := fmt.Sprintf("config_version=1\n[defaults]\nvalidation=true\ndeduplicate_models=true\noutput_dir='out'\nmodel_name_suffix='Dto'\n[[specs]]\nname='a'\ninput=%q\npackage_name='org.example.dedupconfig'\n%s\n", input, value)
		if e := os.WriteFile(cfg, []byte(text), 0600); e != nil {
			t.Fatal(e)
		}
	}
	args := []string{"generate", "--config", cfg, "--spec", "a", "--diagnostic-format=json"}
	write("")
	result(t, invoke(t, repo, args...), 0)
	file := filepath.Join(root, "org/example/dedupconfig/api/models/BetaResponseDto.kt")
	text, _ := os.ReadFile(file)
	if !strings.Contains(string(text), "public typealias BetaResponseDto") {
		t.Fatal("defaults flag missing", string(text))
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("check writes")
	}
	reset := append(append([]string{}, args...), "--deduplicate-models=false")
	result(t, invoke(t, repo, append(reset, "--check")...), 3)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("check reset writes")
	}
	result(t, invoke(t, repo, reset...), 0)
	text, _ = os.ReadFile(file)
	if !strings.Contains(string(text), "public data class BetaResponseDto") {
		t.Fatal("explicitfalse lost", string(text))
	}
	write("deduplicate_models=false")
	result(t, invoke(t, repo, args...), 0)
	text, _ = os.ReadFile(file)
	if strings.Contains(string(text), "public typealias") {
		t.Fatal("spec false lost")
	}
	result(t, invoke(t, repo, append(args, "--deduplicate-models")...), 0)
	before = snapshot(t, root)
	write("deduplicate_models='sensitive'")
	r := result(t, invoke(t, repo, args...), 1)
	if r.Diagnostics[0].Code != "E_CONFIG_INVALID" || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal(r, "invalid config changed output")
	}
	for _, extra := range [][]string{{"--deduplicate-models=maybe"}, {"--deduplicate-models", "--deduplicate-models"}, {"--deduplicate-models="}} {
		result(t, invoke(t, repo, append(deduplicateArgs(root), extra...)...), 2)
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("usage writes")
		}
	}
	result(t, invoke(t, repo, "download", "--config", cfg, "--deduplicate-models", "--diagnostic-format=json"), 2)
}
