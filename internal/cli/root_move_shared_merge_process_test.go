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
	"time"
)

func TestRootMoveSharedMergeNativeProfilesReportsAndPostMove(t *testing.T) {
	for _, locked := range []bool{false, true} {
		for _, style := range []string{"data_class", "plain_class"} {
			for _, selected := range []bool{false, true} {
				base, cfg, src, dst, text := mergeConfigFixtureForShared(t, locked, style, true)
				for path := range snapshot(t, dst) {
					if filepath.Ext(path) == ".kt" {
						full := filepath.Join(dst, path)
						if e := os.Chmod(full, 0640); e != nil {
							t.Fatal(e)
						}
						stamp := time.Unix(1200000000, 0)
						if e := os.Chtimes(full, stamp, stamp); e != nil {
							t.Fatal(e)
						}
					}
				}
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
							SharedFiles  []string `json:"shared_target_files"`
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
					if len(raw.Migration.SharedFiles) == 0 || len(raw.Migration.SharedFiles) != len(raw.Migration.Files) {
						t.Fatal("shared path report missing", string(out.Stdout))
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
						want := a[path]
						if _, shared := b[path]; shared && (!selected || owner == "alpha") {
							want = b[path]
						}
						if !reflect.DeepEqual(want, root[path]) {
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
