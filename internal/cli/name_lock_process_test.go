package cli_test

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
)

func TestNameLockCLIWholeEnvelopeControl(t *testing.T) {
	TestHoistWholeEnvelopeDefaultControl(t)
	TestNameLockCLIDefaultConfigControl(t)
}
func TestNameLockCLIDefaultConfigControl(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "generator.toml")
	input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
	put(t, cfg, []byte("config_version = 1\n[defaults]\ninput = "+strconv.Quote(input)+"\noutput_dir = \"generated\"\npackage_name = \"org.example.lockcontrol\"\nmodel_name_suffix = \"Dto\"\nvalidation = true\nhoist_shared_shapes = true\n[[specs]]\nname = \"alpha\"\n"))
	args := []string{"generate", "--config", cfg, "--diagnostic-format", "json"}
	result(t, invoke(t, repo, args...), 0)
	var m map[string]json.RawMessage
	if e := json.Unmarshal(read(t, filepath.Join(root, "generated/.openapi-kt-manifest.json")), &m); e != nil || string(m["version"]) != "2" || len(m) != 3 {
		t.Fatal("defaultconfig envelope", e)
	}
	result(t, invoke(t, repo, append(args, "--check")...), 0)
}
func TestNameLockCLIDesiredSuccess(t *testing.T) {
	root := t.TempDir()
	args := append(deduplicateArgs(root), "--hoist-shared-shapes", "--lock-hoisted-names")
	result(t, invoke(t, repo, args...), 0)
}
