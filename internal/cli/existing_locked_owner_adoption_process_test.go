package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func existingDonorFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	base := t.TempDir()
	input, cfg := splitAdoptionInputs(t, base)
	root := filepath.Join(base, "generated")
	result(t, invoke(t, base, "generate", "--input", input, "--output", root, "--package-name=org.example.split", "--model-name-suffix=Dto", "--api-client-name=SplitClient", "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"), 0)
	donorCfg := filepath.Join(base, "donor.toml")
	defaults, _, _ := strings.Cut(string(read(t, cfg)), "[[specs]]")
	put(t, donorCfg, []byte(defaults+"[[specs]]\nname='catalog'\ninput='source.json'\n"))
	result(t, invoke(t, base, "generate", "--config", donorCfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"), 0)
	return base, cfg, root, donorCfg
}

func TestExistingV4OwnerAdoptionActualProjectedSubsets(t *testing.T) {
	base, cfg, root, donorCfg := existingDonorFixture(t)
	for _, owner := range []string{"api", "models"} {
		before := snapshot(t, root)
		run := []string{"generate", "--config", cfg, "--spec=" + owner, "--adopt-locked-owner", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"}
		checked := invoke(t, base, append(append([]string{}, run...), "--check")...)
		result(t, checked, 3)
		validated := invoke(t, base, lifecycleValidate(run)...)
		result(t, validated, 0)
		var reports [2]struct {
			Adoptions []struct {
				Owner  string               `json:"owner"`
				From   string               `json:"from_owner"`
				Before int                  `json:"before_version"`
				After  int                  `json:"after_version"`
				Source []nameLockTestSymbol `json:"source_symbols"`
			} `json:"locked_owner_adoptions"`
		}
		for i, out := range []invocation{checked, validated} {
			if err := json.Unmarshal(out.Stdout, &reports[i]); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(reports[0], reports[1]) || len(reports[0].Adoptions) != 1 {
			t.Fatal("unbound readonly report", reports)
		}
		m := reports[0].Adoptions[0]
		if m.Owner != owner || m.From != "catalog" || m.Before != 4 || m.After != 4 || len(m.Source) == 0 {
			t.Fatal("incomplete donor report", m)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("proposal wrote metadata")
		}
		result(t, invoke(t, base, run...), 0)
		after := snapshot(t, root)
		for path, old := range before {
			if path != ".openapi-kt-manifest.json" && !reflect.DeepEqual(old, after[path]) {
				t.Fatal("old peer bytes/mode/mtime changed", path)
			}
		}
		result(t, invoke(t, base, append(append([]string{}, run...), "--check")...), 0)
		result(t, invoke(t, base, "generate", "--config", cfg, "--spec="+owner, "--check", "--diagnostic-format=json"), 0)
		result(t, invoke(t, base, "generate", "--config", donorCfg, "--spec=catalog", "--check", "--diagnostic-format=json"), 0)
	}
	var manifest struct {
		Owners    map[string][]string
		NameLocks map[string]string `json:"name_locks"`
	}
	if err := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Owners) != 3 || len(manifest.Owners["models"]) != 1 || len(manifest.Owners["api"]) <= 1 || len(manifest.Owners["api"]) >= len(manifest.Owners["catalog"]) {
		t.Fatal("not genuine projected subsets", manifest)
	}
	var donor, api, models struct {
		Entries []struct {
			Occurrences []string `json:"occurrences"`
		} `json:"entries"`
	}
	for name, target := range map[string]any{"catalog": &donor, "api": &api, "models": &models} {
		if err := json.Unmarshal(read(t, filepath.Join(root, namedLockPath(name))), target); err != nil {
			t.Fatal(err)
		}
	}
	if len(donor.Entries) == 0 || len(api.Entries) == 0 || len(models.Entries) != 0 {
		t.Fatal("locks not projected", donor, api, models)
	}
}

type nameLockTestSymbol map[string]any

func TestExistingV4OwnerAdoptionUsageAndRejections(t *testing.T) {
	base, cfg, root, _ := existingDonorFixture(t)
	run := []string{"generate", "--config", cfg, "--spec=api", "--adopt-locked-owner", "--diagnostic-format=json"}
	for _, tail := range [][]string{{"--adopt-locked-owner-from="}, {"--adopt-locked-owner-from=api"}, {"--adopt-locked-owner-from=../catalog"}, {"--adopt-locked-owner-from=catalog", "--adopt-locked-owner-from=catalog"}, {"--adopt-locked-owner-from=catalog", "--rename-owner=x:api"}, {"--adopt-locked-owner-from=catalog", "--name-lock-action=reset"}} {
		fails(t, invoke(t, base, append(append([]string{}, run...), tail...)...), 2, "E_USAGE")
	}
	fails(t, invoke(t, base, "generate", "--config", cfg, "--spec=api", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"), 2, "E_USAGE")
	fails(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"), 2, "E_USAGE")
	before := snapshot(t, root)
	for _, args := range [][]string{append(append([]string{}, run...), "--adopt-locked-owner-from=absent"), append(append([]string{}, run...), "--adopt-locked-owner-from=catalog", "--package-name=org.example.changed"), append(append([]string{}, run...), "--adopt-locked-owner-from=catalog", "--hoist-shared-shapes=false", "--lock-hoisted-names=false")} {
		out := invoke(t, base, args...)
		if out.Code == 0 || out.Code == 3 {
			t.Fatal("invalid donor/profile accepted", out)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("failure modified root")
		}
	}
	for _, command := range []string{"generate", "validate"} {
		out := invoke(t, base, command, "--help")
		if !strings.Contains(string(out.Stdout), "--adopt-locked-owner-from <OWNER>") {
			t.Fatal("missing donor help")
		}
	}
}

func TestExistingV4OwnerAdoptionInvalidSourceBeforeInput(t *testing.T) {
	base, cfg, root, _ := existingDonorFixture(t)
	put(t, cfg, []byte(strings.Replace(string(read(t, cfg)), "input='api.json'", "input='missing.json'", 1)))
	before := snapshot(t, root)
	run := []string{"generate", "--config", cfg, "--spec=api", "--adopt-locked-owner", "--diagnostic-format=json"}
	fails(t, invoke(t, base, append(append([]string{}, run...), "--adopt-locked-owner-from=absent")...), 4, "E_LOCKED_OWNER_ADOPTION_INVALID")
	fails(t, invoke(t, base, append(append([]string{}, run...), "--adopt-locked-owner-from=catalog", "--package-name=org.example.changed")...), 1, "E_NAME_LOCK_PROFILE")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("invalid pre-input source changed state")
	}
}

func TestExistingV4OwnerAdoptionExplicitDonorMetadataOnly(t *testing.T) {
	base, cfg, root, _ := lockedAdoptionFixture(t)
	result(t, invoke(t, base, "generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	nextCfg := filepath.Join(base, "attach.toml")
	put(t, nextCfg, []byte(strings.Replace(string(read(t, cfg)), "name='catalog'", "name='consumer'", 1)))
	run := []string{"generate", "--config", nextCfg, "--spec=consumer", "--adopt-locked-owner", "--adopt-locked-owner-from=catalog", "--diagnostic-format=json"}
	textReport := invoke(t, base, append(append([]string{}, run[:len(run)-1]...), "--check")...)
	if textReport.Code != 3 || !strings.Contains(string(textReport.Stdout), "allocation donor catalog") {
		t.Fatal("text report lacks donor", textReport)
	}
	result(t, invoke(t, base, append(append([]string{}, run...), "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("v4 donor proposal wrote old inventory")
	}
	result(t, invoke(t, base, run...), 0)
	after := snapshot(t, root)
	for path, old := range before {
		if path != ".openapi-kt-manifest.json" && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("v4 donor adoption modified existing file/lock attributes", path)
		}
	}
	if after[namedLockPath("consumer")].Bytes != before[namedLockPath("catalog")].Bytes {
		t.Fatal("unchanged donor raw lock not reused")
	}
	result(t, invoke(t, base, append(append([]string{}, run...), "--check")...), 0)
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("v4 donor replay mutated output")
	}
	result(t, invoke(t, base, "generate", "--config", cfg, "--spec=catalog", "--check", "--diagnostic-format=json"), 0)
	result(t, invoke(t, base, "generate", "--config", nextCfg, "--spec=consumer", "--check", "--diagnostic-format=json"), 0)
}
