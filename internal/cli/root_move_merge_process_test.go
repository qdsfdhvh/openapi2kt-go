package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mergeConfigFixture(t testing.TB, locked bool, style string) (string, string, string, string, string) {
	return mergeConfigFixtureForShared(t, locked, style, false)
}

func mergeConfigFixtureForShared(t testing.TB, locked bool, style string, shared bool) (string, string, string, string, string) {
	t.Helper()
	base := t.TempDir()
	names := []string{"alpha", "beta", "gamma"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, locked, false, false)
	text := strings.Replace(string(read(t, cfg)), "[defaults]", "[defaults]\nmodel_style='"+style+"'", 1)
	text = strings.Replace(text, "name=\"gamma\"", "name=\"gamma\"\noutput_dir='target'", 1)
	if shared {
		for _, name := range names {
			text = strings.ReplaceAll(text, "org.example.lock"+name, "org.example.sharedroot")
		}
	}
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	return base, cfg, filepath.Join(base, "generated"), filepath.Join(base, "target"), text
}

func TestRootMoveMergeNativeProfilesReportsAndPostMove(t *testing.T) {
	for _, locked := range []bool{false, true} {
		for _, style := range []string{"data_class", "plain_class"} {
			for _, selected := range []bool{false, true} {
				base, cfg, src, dst, text := mergeConfigFixture(t, locked, style)
				a, b := snapshot(t, src), snapshot(t, dst)
				targetManifest := read(t, filepath.Join(dst, ".openapi-kt-manifest.json"))
				hash := sha256.Sum256(targetManifest)
				// An existing target peer is complete storage control, not a source input.
				original := read(t, filepath.Join(base, "gamma.json"))
				if e := os.Remove(filepath.Join(base, "gamma.json")); e != nil {
					t.Fatal(e)
				}
				args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst}
				if selected {
					args = append(args, "--spec=alpha")
				}
				checked := invoke(t, base, append(append([]string{}, args...), "--check")...)
				result(t, checked, 3)
				validated := invoke(t, base, lifecycleValidate(args)...)
				result(t, validated, 0)
				if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
					t.Fatal("readonly merge wrote")
				}
				applied := invoke(t, base, args...)
				result(t, applied, 0)
				for _, out := range []invocation{checked, validated, applied} {
					var raw struct {
						Migration struct {
							TargetHash   string   `json:"target_manifest_sha256"`
							TargetOwners []string `json:"target_owners"`
							Version      int
							Files        []struct{ Before, After, SHA256 string }
						} `json:"output_root_migration"`
						Specs []struct{ Name string }
					}
					if e := json.Unmarshal(out.Stdout, &raw); e != nil {
						t.Fatal(e)
					}
					if raw.Migration.TargetHash != hex.EncodeToString(hash[:]) || !reflect.DeepEqual(raw.Migration.TargetOwners, []string{"gamma"}) || len(raw.Specs) != 2 || bytes.Contains(out.Stdout, []byte(base)) {
						t.Fatal("lost source/target report", string(out.Stdout))
					}
					version := 2
					if locked {
						version = 4
					}
					if raw.Migration.Version != version {
						t.Fatal("version changed")
					}
					for _, f := range raw.Migration.Files {
						sum := sha256.Sum256(read(t, filepath.Join(dst, f.After)))
						if f.Before != f.After || hex.EncodeToString(sum[:]) != f.SHA256 {
							t.Fatal("report does not bind moved bytes")
						}
					}
				}
				afterA, afterB := snapshot(t, src), snapshot(t, dst)
				for path, entry := range b {
					if path != ".openapi-kt-manifest.json" && !reflect.DeepEqual(entry, afterB[path]) {
						t.Fatal("old target changed", path)
					}
				}
				var source struct{ Owners map[string][]string }
				if e := json.Unmarshal([]byte(a[".openapi-kt-manifest.json"].Bytes), &source); e != nil {
					t.Fatal(e)
				}
				for owner, paths := range source.Owners {
					root := afterB
					if selected && owner != "alpha" {
						root = afterA
					}
					for _, path := range paths {
						if !reflect.DeepEqual(a[path], root[path]) {
							t.Fatal("source bytes/attrs changed", owner, path)
						}
					}
				}
				if selected {
					text = strings.Replace(text, "name=\"alpha\"", "name=\"alpha\"\noutput_dir='target'", 1)
				} else {
					text = strings.Replace(text, "output_dir='generated'", "output_dir='target'", 1)
				}
				put(t, cfg, []byte(text))
				put(t, filepath.Join(base, "gamma.json"), original)
				ordinary := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
				result(t, invoke(t, base, append(append([]string{}, ordinary...), "--check")...), 0)
				result(t, invoke(t, base, ordinary...), 0)
				if !reflect.DeepEqual(afterA, snapshot(t, src)) || !reflect.DeepEqual(afterB, snapshot(t, dst)) {
					t.Fatal("ordinary generation rewrote merged inventories")
				}
			}
		}
	}
}

func TestRootMoveMergeNativeWrongDeclaredPeerAndSourceChangeRefuse(t *testing.T) {
	for _, kind := range []string{"unknown-target-owner", "target-peer-at-source", "source-change"} {
		base, cfg, src, dst, text := mergeConfigFixture(t, kind != "source-change", "data_class")
		if kind == "unknown-target-owner" {
			text = strings.Replace(text, "name=\"gamma\"", "name=\"delta\"", 1)
		}
		if kind == "target-peer-at-source" {
			text = strings.Replace(text, "output_dir='target'", "output_dir='generated'", 1)
		}
		put(t, cfg, []byte(text))
		if kind == "source-change" {
			(project{Input: filepath.Join(base, "alpha.json")}).change(t, func(doc map[string]any) {
				doc["components"] = map[string]any{"schemas": map[string]any{"Extra": map[string]any{"type": "object", "properties": map[string]any{"label": map[string]any{"type": "string"}}}}}
			})
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		code := "E_OUTPUT_ROOT_MOVE_INVALID"
		if kind == "source-change" {
			code = "E_OUTPUT_ROOT_MOVE_CHANGED"
		}
		fails(t, invoke(t, base, "generate", "--config", cfg, "--output", dst, "--move-output-from", src, "--diagnostic-format=json"), 4, code)
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("failed merge wrote", kind)
		}
	}
}
