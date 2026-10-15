package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNameLockConfigNativeMultiRootAllGenerationBeforeCommit(t *testing.T) {
	base := t.TempDir()
	a := lockInput(t, base)
	b := filepath.Join(base, "beta.json")
	put(t, b, read(t, a))
	cfg := filepath.Join(base, "config.toml")
	put(t, cfg, []byte("config_version=1\n[defaults]\nvalidation=true\nmodel_name_suffix='Dto'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='alpha'\ninput='input.json'\noutput_dir='alpha'\npackage_name='org.example.lockalpha'\n[[specs]]\nname='beta'\ninput='beta.json'\noutput_dir='beta'\npackage_name='org.example.lockbeta'\n"))
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	ar, br := filepath.Join(base, "alpha"), filepath.Join(base, "beta")
	beforeA, beforeB := snapshot(t, ar), snapshot(t, br)
	lockGrow(t, a)
	var doc, paths map[string]json.RawMessage
	_ = json.Unmarshal(read(t, b), &doc)
	_ = json.Unmarshal(doc["paths"], &paths)
	delete(paths, "/beta")
	doc["paths"], _ = json.Marshal(paths)
	bad, _ := json.Marshal(doc)
	put(t, b, bad)
	r := fails(t, invoke(t, base, args...), 1, "E_NAME_LOCK_REMOVED")
	_ = r
	if !reflect.DeepEqual(beforeA, snapshot(t, ar)) || !reflect.DeepEqual(beforeB, snapshot(t, br)) {
		t.Fatal("later spec's failed generation committed earlier root/state")
	}
	put(t, b, read(t, filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")))
	lockGrow(t, b)
	result(t, invoke(t, base, append(args, "--check")...), 3)
	if !reflect.DeepEqual(beforeA, snapshot(t, ar)) || !reflect.DeepEqual(beforeB, snapshot(t, br)) {
		t.Fatal("batchcheck wrote")
	}
	result(t, invoke(t, base, args...), 0)
	// Owned Kotlin failure in the later root rejects the whole transaction.
	afterA := snapshot(t, ar)
	var owned string
	for p := range snapshot(t, br) {
		if strings.HasSuffix(p, ".kt") {
			owned = p
			break
		}
	}
	put(t, filepath.Join(br, owned), []byte("synthetic manually edited Kotlin"))
	afterB := snapshot(t, br)
	fails(t, invoke(t, base, args...), 4, "E_OUTPUT_MODIFIED")
	if !reflect.DeepEqual(afterA, snapshot(t, ar)) || !reflect.DeepEqual(afterB, snapshot(t, br)) {
		t.Fatal("outputfailure changed prior metadata")
	}
}
func TestNameLockCLIStorageGuardsNativePreserve(t *testing.T) {
	for _, kind := range []string{"missing", "limit", "unowned"} {
		base := t.TempDir()
		input := lockInput(t, base)
		root := filepath.Join(base, "out")
		args := lockArgs(input, root)
		state := filepath.Join(root, ".openapi-kt-names.json")
		code := "E_NAME_LOCK_MISSING"
		if kind == "unowned" {
			if e := os.Mkdir(root, 0700); e != nil {
				t.Fatal(e)
			}
			put(t, state, []byte("synthetic unowned"))
			code = "E_OUTPUT_COLLISION"
		} else {
			result(t, invoke(t, base, args...), 0)
			if kind == "missing" {
				if e := os.Remove(state); e != nil {
					t.Fatal(e)
				}
			} else {
				put(t, state, []byte(strings.Repeat(" ", (1<<20)+1)))
				code = "E_OUTPUT_LIMIT"
			}
		}
		before := snapshot(t, root)
		for _, tail := range [][]string{nil, {"--check"}} {
			fails(t, invoke(t, base, append(args, tail...)...), 4, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("storagefailure wrote")
			}
		}
	}
}
