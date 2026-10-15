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

func lockedAdoptionFixture(t testing.TB) (string, string, string, []string) {
	t.Helper()
	base := t.TempDir()
	input := filepath.Join(base, "catalog.json")
	put(t, input, read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
	root := filepath.Join(base, "generated")
	direct := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.adoption", "--model-name-suffix=Dto", "--hoist-shared-shapes", "--qualify-hoisted-names", "--lock-hoisted-names", "--validation", "--diagnostic-format=json"}
	result(t, invoke(t, base, direct...), 0)
	cfg := filepath.Join(base, "adopt.toml")
	put(t, cfg, []byte("config_version=0x1\ndefaults={output_dir='generated',package_name='org.example.adoption',model_name_suffix='Dto',hoist_shared_shapes=true,qualify_hoisted_names=true,lock_hoisted_names=true,validation=true}\nspecs=[{name='catalog',input='catalog.json'}]\n"))
	return base, cfg, root, direct
}

func TestLockedOwnerAdoptionNativeMetadataOnlyAndReport(t *testing.T) {
	base, cfg, root, direct := lockedAdoptionFixture(t)
	put(t, filepath.Join(root, "Notes.kt"), []byte("// synthetic handwritten control"))
	put(t, filepath.Join(root, namedLockPath("foreign")), []byte("foreign metadata control"))
	before := snapshot(t, root)
	run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	check := invoke(t, base, append(run, "--check")...)
	result(t, check, 3)
	validated := invoke(t, base, lifecycleValidate(run)...)
	result(t, validated, 0)
	var reports [2]struct {
		Adoptions []struct {
			Owner          string `json:"owner"`
			BeforeVersion  int    `json:"before_version"`
			AfterVersion   int    `json:"after_version"`
			ManifestBefore string `json:"manifest_before_sha256"`
			ManifestAfter  string `json:"manifest_after_sha256"`
			NameLock       struct {
				Before  string           `json:"before_sha256"`
				After   string           `json:"after_sha256"`
				Symbols []map[string]any `json:"symbols"`
			} `json:"name_lock"`
		} `json:"locked_owner_adoptions"`
	}
	for i, out := range []invocation{check, validated} {
		if e := json.Unmarshal(out.Stdout, &reports[i]); e != nil {
			t.Fatal(e)
		}
	}
	if !reflect.DeepEqual(reports[0], reports[1]) || len(reports[0].Adoptions) != 1 {
		t.Fatal("check/validate report binding", reports)
	}
	r := reports[0].Adoptions[0]
	if r.Owner != "catalog" || r.BeforeVersion != 3 || r.AfterVersion != 4 || len(r.ManifestBefore) != 64 || len(r.ManifestAfter) != 64 || r.NameLock.Before != r.NameLock.After || len(r.NameLock.Symbols) == 0 {
		t.Fatal("incomplete report", r)
	}
	for _, s := range r.NameLock.Symbols {
		if s["classifier"] == "" || s["descriptor"] == "" {
			t.Fatal(s)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("read-only migration wrote output")
	}
	result(t, invoke(t, base, run...), 0)
	after := snapshot(t, root)
	for path, file := range before {
		if path != ".openapi-kt-manifest.json" && path != ".openapi-kt-names.json" && !reflect.DeepEqual(file, after[path]) {
			t.Fatal("migration changed source/foreign/control attrs", path)
		}
	}
	if after[namedLockPath("catalog")].Bytes != before[".openapi-kt-names.json"].Bytes {
		t.Fatal("raw lock bytes lost")
	}
	if _, e := os.Stat(filepath.Join(root, ".openapi-kt-names.json")); !os.IsNotExist(e) {
		t.Fatal("anonymous duplicate", e)
	}
	for _, args := range [][]string{append(run, "--check"), lifecycleValidate(run), run, {"generate", "--config", cfg, "--spec=catalog", "--check", "--diagnostic-format=json"}} {
		result(t, invoke(t, base, args...), 0)
	}
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("post adoption no-op changed output")
	}
	fails(t, invoke(t, base, append(direct, "--check")...), 4, "E_MANIFEST_VERSION")
}

func TestLockedOwnerAdoptionNativeClosedSelectionAndNoFreshRoot(t *testing.T) {
	base, cfg, root, _ := lockedAdoptionFixture(t)
	before := snapshot(t, root)
	common := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	for _, tail := range [][]string{{"--adopt-locked-owner"}, {"--adopt-locked-owner=true"}, {"--adopt-legacy"}, {"--reconcile"}, {"--rename-owner=x:catalog"}, {"--name-lock-action=reset"}, {"--name-lock-action=disable"}, {"--move-output-from=unused", "--output=unused"}} {
		fails(t, invoke(t, base, append(append([]string(nil), common...), tail...)...), 2, "E_USAGE")
	}
	fails(t, invoke(t, base, "generate", "--config", cfg, "--adopt-locked-owner", "--diagnostic-format=json"), 2, "E_USAGE")
	fails(t, invoke(t, base, "validate", "--input=unused", "--adopt-locked-owner", "--diagnostic-format=json"), 2, "E_USAGE")
	for _, command := range []string{"generate", "validate"} {
		out := invoke(t, base, command, "--help")
		if out.Code != 0 || !bytes.Contains(out.Stdout, []byte("--adopt-locked-owner")) {
			t.Fatal("missing help")
		}
	}
	original := read(t, cfg)
	put(t, cfg, append(append([]byte(nil), original...), []byte("# two specs are outside this bounded action\n")...))
	put(t, cfg, []byte(strings.ReplaceAll(string(original), "specs=[{name='catalog',input='catalog.json'}]", "specs=[{name='catalog',input='catalog.json'},{name='peer',input='missing.json'}]")))
	fails(t, invoke(t, base, common...), 4, "E_LOCKED_OWNER_ADOPTION_INVALID")
	put(t, cfg, []byte(strings.ReplaceAll(string(original), "lock_hoisted_names=true", "lock_hoisted_names=false")))
	fails(t, invoke(t, base, common...), 4, "E_LOCKED_OWNER_ADOPTION_INVALID")
	put(t, cfg, original)
	fresh := filepath.Join(base, "fresh")
	fails(t, invoke(t, base, append(common, "--output="+fresh)...), 4, "E_LOCKED_OWNER_ADOPTION_INVALID")
	if _, e := os.Stat(fresh); !os.IsNotExist(e) {
		t.Fatal("created changed root", e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("guard changed old root")
	}
}
