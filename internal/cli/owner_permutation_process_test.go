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

func TestOwnerPermutationNativeReadOnlyApplyReportsAndNormalReplay(t *testing.T) {
	for _, locked := range []bool{false, true} {
		base := t.TempDir()
		lifecycleConfigInputs(t, base, []string{"alpha", "beta"})
		cfg := lifecycleConfig(t, base, []string{"alpha", "beta"}, locked, false, true)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		root := filepath.Join(base, "generated")
		before := snapshot(t, root)
		renameConfig(t, cfg, "alpha", "temporary")
		renameConfig(t, cfg, "beta", "alpha")
		renameConfig(t, cfg, "temporary", "beta")
		run := append(append([]string(nil), args...), "--rename-owner=alpha:beta", "--rename-owner=beta:alpha")
		// A partial selection cannot claim an occupied target whose old owner
		// is outside the selected mapping, even before a missing input is read.
		input := filepath.Join(base, "alpha.json")
		backup := filepath.Join(base, "alpha-saved.json")
		if e := os.Rename(input, backup); e != nil {
			t.Fatal(e)
		}
		fails(t, invoke(t, base, append(append([]string(nil), args...), "--spec=beta", "--rename-owner=alpha:beta")...), 4, "E_OWNER_RENAME_INVALID")
		if e := os.Rename(backup, input); e != nil {
			t.Fatal(e)
		}
		checked := invoke(t, base, append(run, "--check")...)
		result(t, checked, 3)
		validated := invoke(t, base, lifecycleValidate(run)...)
		result(t, validated, 0)
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("Permutation check/validate changed output")
		}
		committed := invoke(t, base, run...)
		result(t, committed, 0)
		var cr, ar map[string]json.RawMessage
		if e := json.Unmarshal(checked.Stdout, &cr); e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal(committed.Stdout, &ar); e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(cr["owner_renames"], ar["owner_renames"]) {
			t.Fatal("Permutation report changed between check and apply")
		}
		after := snapshot(t, root)
		for path, old := range before {
			if strings.HasSuffix(path, ".kt") && !reflect.DeepEqual(old, after[path]) {
				t.Fatal("Native owner swap rewrote Kotlin bytes/mode/mtime", path)
			}
		}
		var prior, next struct {
			Version   int
			Files     map[string]string
			Owners    map[string][]string
			NameLocks map[string]string `json:"name_locks"`
		}
		if e := json.Unmarshal([]byte(before[".openapi-kt-manifest.json"].Bytes), &prior); e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &next); e != nil {
			t.Fatal(e)
		}
		if prior.Version != next.Version || !reflect.DeepEqual(prior.Files, next.Files) || !reflect.DeepEqual(prior.Owners["alpha"], next.Owners["beta"]) || !reflect.DeepEqual(prior.Owners["beta"], next.Owners["alpha"]) {
			t.Fatal("Permutation lost source membership/version/files", prior, next)
		}
		if locked {
			for old, target := range map[string]string{"alpha": "beta", "beta": "alpha"} {
				if before[namedLockPath(old)].Bytes != after[namedLockPath(target)].Bytes || prior.NameLocks[old] != next.NameLocks[target] {
					t.Fatal("Permutation lost raw lock payload", old, target)
				}
			}
		}
		result(t, invoke(t, base, append(args, "--check")...), 0)
		// Reapplying the old permutation is a new request against the new owner
		// inventory, so these different source candidates must be rejected.
		stable := snapshot(t, root)
		if locked {
			fails(t, invoke(t, base, append(run, "--check")...), 1, "E_NAME_LOCK_PROFILE")
		} else {
			fails(t, invoke(t, base, append(run, "--check")...), 4, "E_OWNER_RENAME_CHANGED")
		}
		if !reflect.DeepEqual(stable, snapshot(t, root)) {
			t.Fatal("Rejected old permutation changed the new inventory")
		}
	}
}
