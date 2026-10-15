package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This consumer fixture is also compiled by the HTTP namespace contract suite.
// Rename must preserve the current emitter's aliases and bytes through full TOML
// syntax; it must not substitute output from a pre-namespace-fix generator.
func TestOwnerRenameNativeFullTOMLNamespaceConsumerParity(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "namespace.json")
	put(t, input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "namespace-collisions.json")))
	cfg := filepath.Join(base, "rename.toml")
	put(t, cfg, []byte("config_version=0x1\ndefaults={output_dir='generated', package_name='org.example.namespaces', model_name_suffix='Dto', hoist_shared_shapes=true, lock_hoisted_names=true}\nspecs=[{name='old', input='namespace.json'}]\n"))
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	root := filepath.Join(base, "generated")
	before := snapshot(t, root)

	// Direct generation uses the accepted HTTP consumer profile without locking;
	// config locking and rename must retain those exact current-emitter bytes.
	control := filepath.Join(base, "control")
	result(t, invoke(t, base, "generate", "--input", input, "--output", control, "--package-name=org.example.namespaces", "--model-name-suffix=Dto", "--diagnostic-format=json"), 0)
	controlFiles := snapshot(t, control)
	for path, file := range before {
		if strings.HasSuffix(path, ".kt") && file.Bytes != controlFiles[path].Bytes {
			t.Fatal("configured full TOML output differs from direct current emitter", path)
		}
	}
	apiPath := filepath.Join(root, "org", "example", "namespaces", "api", "ShadowsApi.kt")
	api := string(read(t, apiPath))
	for _, alias := range []string{"import io.ktor.http.HttpMethod as OpenapiKtHttpMethodImport", "import kotlinx.coroutines.flow.flow as OpenapiKtFlowImport"} {
		if !strings.Contains(api, alias) {
			t.Fatal("current namespace alias absent", alias)
		}
	}

	// Change only the owner while using another equivalent full TOML spelling.
	put(t, cfg, []byte("config_version=+1\n[\"defaults\"]\n\"output_dir\"='generated'\npackage_name='org.example.namespaces'\nmodel_name_suffix='Dto'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[\"specs\"]]\n\"name\"='new'\ninput=\"\"\"namespace.json\"\"\"\n"))
	rename := append(append([]string(nil), args...), "--spec=new", "--rename-owner=old:new")
	result(t, invoke(t, base, append(rename, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(rename)...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("full TOML consumer rename preflight wrote output")
	}
	result(t, invoke(t, base, rename...), 0)
	after := snapshot(t, root)
	for path, file := range before {
		if strings.HasSuffix(path, ".kt") && !reflect.DeepEqual(file, after[path]) {
			t.Fatal("rename changed namespace consumer bytes/modes/mtimes", path)
		}
	}
	if before[namedLockPath("old")].Bytes != after[namedLockPath("new")].Bytes {
		t.Fatal("rename changed existing lock bytes")
	}
	if _, e := os.Stat(filepath.Join(root, namedLockPath("old"))); !os.IsNotExist(e) {
		t.Fatal("old owner metadata survived", e)
	}
	var m struct {
		Version int
		Owners  map[string][]string
	}
	if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil || m.Version != 4 || len(m.Owners) != 1 || len(m.Owners["new"]) == 0 {
		t.Fatal("full TOML consumer owner transfer", m, e)
	}
	result(t, invoke(t, base, append(args, "--check")...), 0)
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("post-rename full TOML check changed consumer")
	}
}
