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

func TestOwnerRenameNativeFullApplyAndIdentityMix(t *testing.T) {
	for _, multi := range []bool{false, true} {
		for _, identity := range []bool{false, true} {
			base := t.TempDir()
			names := []string{"alpha", "beta"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, true, multi, !multi)
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, base, args...), 0)
			roots := []string{filepath.Join(base, "generated")}
			if multi {
				roots = []string{filepath.Join(base, "alpha-output"), filepath.Join(base, "beta-output")}
			}
			before := map[string]map[string]snap{}
			for _, root := range roots {
				before[root] = snapshot(t, root)
			}
			next := "gamma"
			if identity {
				next = "alpha"
			} else {
				renameConfig(t, cfg, "alpha", next)
			}
			renameConfig(t, cfg, "beta", "delta")
			run := append(append([]string(nil), args...), "--rename-owner=alpha:"+next, "--rename-owner=beta:delta")
			checked := invoke(t, base, append(run, "--check")...)
			result(t, checked, 3)
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
				t.Fatal("full report changed")
			}
			var report []struct {
				New string `json:"new_owner"`
			}
			if e := json.Unmarshal(ar["owner_renames"], &report); e != nil || len(report) != 2 || report[0].New >= report[1].New {
				t.Fatal("full report order", report, e)
			}
			for _, root := range roots {
				after := snapshot(t, root)
				for path, file := range before[root] {
					if strings.HasSuffix(path, ".kt") && !reflect.DeepEqual(file, after[path]) {
						t.Fatal("full apply changed Kotlin")
					}
				}
				var m struct {
					Owners    map[string][]string
					NameLocks map[string]string `json:"name_locks"`
				}
				if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil {
					t.Fatal(e)
				}
				for old, newName := range map[string]string{"alpha": next, "beta": "delta"} {
					prior, participates := before[root][namedLockPath(old)]
					if !participates {
						continue
					}
					if len(m.Owners[newName]) == 0 || m.NameLocks[newName] == "" || prior.Bytes != after[namedLockPath(newName)].Bytes {
						t.Fatal("full state transfer", old, newName, m)
					}
					if old != newName {
						if _, e := os.Stat(filepath.Join(root, namedLockPath(old))); !os.IsNotExist(e) {
							t.Fatal("old payload survived")
						}
					} else if !reflect.DeepEqual(prior, after[namedLockPath(old)]) {
						t.Fatal("identity metadata rewritten")
					}
				}
			}
			result(t, invoke(t, base, append(args, "--check")...), 0)
		}
	}
}

func TestOwnerRenameNativeWarningPolicyMetricsAndUnlockedValidationOverrides(t *testing.T) {
	base := t.TempDir()
	lifecycleConfigInputs(t, base, []string{"alpha", "beta"})
	cfg := lifecycleConfig(t, base, []string{"alpha", "beta"}, false, false, true)
	// Add warning-bearing Raw model to the already accepted HTTP consumer fixture.
	for _, name := range []string{"alpha", "beta"} {
		path := filepath.Join(base, name+".json")
		var doc map[string]any
		if e := json.Unmarshal(read(t, path), &doc); e != nil {
			t.Fatal(e)
		}
		if doc["components"] == nil {
			doc["components"] = map[string]any{}
		}
		components := doc["components"].(map[string]any)
		if components["schemas"] == nil {
			components["schemas"] = map[string]any{}
		}
		schemas(doc)["Raw"] = map[string]any{"type": "object"}
		data, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		put(t, path, data)
	}
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	original := result(t, invoke(t, base, args...), 0)
	if len(original.Warnings) == 0 {
		t.Fatal("missing synthetic warnings")
	}
	root := filepath.Join(base, "generated")
	renameConfig(t, cfg, "alpha", "gamma")
	renameConfig(t, cfg, "beta", "delta")
	run := append(append([]string(nil), args...), "--rename-owner=alpha:gamma", "--rename-owner=beta:delta")
	// Denial must finish before output locks, including a missing coordination file.
	if e := os.Remove(filepath.Join(root, ".openapi-kt.lock")); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	for _, mode := range []string{"apply", "check", "validate"} {
		command := append(append([]string(nil), run...), "--deny-warnings", "--metrics")
		if mode == "check" {
			command = append(command, "--check")
		}
		if mode == "validate" {
			command = lifecycleValidate(command)
		}
		out := invoke(t, base, command...)
		r := warningDenied(t, out)
		if !reflect.DeepEqual(r.Warnings, original.Warnings) {
			t.Fatal("denial lost warning provenance")
		}
		var raw map[string]json.RawMessage
		if e := json.Unmarshal(out.Stdout, &raw); e != nil {
			t.Fatal(e)
		}
		if raw["owner_renames"] != nil {
			t.Fatal("denied candidate report escaped")
		}
		metrics := configMetrics(t, out)
		if _, ok := metrics.PhaseNS["commit"]; ok {
			t.Fatal("denial entered commit")
		}
		if _, ok := metrics.PhaseNS["check"]; ok {
			t.Fatal("denial entered check")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("denial created lock/wrote output")
		}
	}
	allowed := append(append([]string(nil), run...), "--deny-warnings")
	seen := map[string]bool{}
	for _, w := range original.Warnings {
		if !seen[w.Code] {
			allowed = append(allowed, "--allow-warning="+w.Code)
			seen[w.Code] = true
		}
	}
	plain := invoke(t, base, append(allowed, "--check")...)
	result(t, plain, 3)
	measured := invoke(t, base, append(allowed, "--check", "--metrics")...)
	result(t, measured, 3)
	m := configMetrics(t, measured)
	files, bytesCount := int64(0), int64(0)
	for path, file := range before {
		if strings.HasSuffix(path, ".kt") {
			files++
			bytesCount += int64(len(file.Bytes))
		}
	}
	if m.PlannedFiles != files || m.PlannedBytes != bytesCount {
		t.Fatal("metadata/shared counted as new source bytes", m, files, bytesCount)
	}
	var p, q map[string]json.RawMessage
	if e := json.Unmarshal(plain.Stdout, &p); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(measured.Stdout, &q); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(p["owner_renames"], q["owner_renames"]) {
		t.Fatal("metrics changed rename report")
	}
	validation := []string{"validate", "--config", cfg, "--spec=gamma", "--rename-owner=alpha:gamma", "--output", root, "--model-name-suffix=Dto", "--diagnostic-format=json"}
	result(t, invoke(t, base, validation...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("readonly metrics/override wrote")
	}
	result(t, invoke(t, base, append(allowed, "--metrics")...), 0)
}
