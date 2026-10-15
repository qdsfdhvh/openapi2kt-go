package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLockedOwnerAdoptionNativeProfileInventoryAndRecoveryGuards(t *testing.T) {
	for _, kind := range []string{"profile", "style", "partial", "missing", "edited", "lockmissing", "lockedited", "occupied", "recovery", "v1"} {
		t.Run(kind, func(t *testing.T) {
			base, cfg, root, _ := lockedAdoptionFixture(t)
			run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
			code, exit := "E_LOCKED_OWNER_ADOPTION_CHANGED", 4
			var m struct {
				Version int               `json:"version"`
				Files   map[string]string `json:"files"`
			}
			if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil {
				t.Fatal(e)
			}
			var path string
			for p := range m.Files {
				path = p
				break
			}
			switch kind {
			case "profile":
				run = append(run, "--package-name=org.example.other")
				code, exit = "E_NAME_LOCK_PROFILE", 1
			case "style":
				run = append(run, "--model-style=plain_class")
			case "partial":
				run = append(run, "--exclude-api=DefaultApi")
			case "missing":
				if e := os.Remove(filepath.Join(root, path)); e != nil {
					t.Fatal(e)
				}
				code = "E_OUTPUT_MISSING"
			case "edited":
				put(t, filepath.Join(root, path), []byte("// edited"))
				code = "E_OUTPUT_MODIFIED"
			case "lockmissing":
				if e := os.Remove(filepath.Join(root, ".openapi-kt-names.json")); e != nil {
					t.Fatal(e)
				}
				code = "E_NAME_LOCK_MISSING"
			case "lockedited":
				put(t, filepath.Join(root, ".openapi-kt-names.json"), []byte("edited"))
				code = "E_NAME_LOCK_MODIFIED"
			case "occupied":
				put(t, filepath.Join(root, namedLockPath("catalog")), read(t, filepath.Join(root, ".openapi-kt-names.json")))
				code = "E_OUTPUT_COLLISION"
			case "recovery":
				stage := filepath.Join(root, ".openapi-kt-stage-retained")
				if e := os.Mkdir(stage, 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(stage, "transaction.json"), []byte("malformed journal"))
				code = "E_RECOVERY_REQUIRED"
			case "v1":
				m.Version = 1
				data, _ := json.Marshal(m)
				put(t, filepath.Join(root, ".openapi-kt-manifest.json"), data)
				code = "E_LOCKED_OWNER_ADOPTION_INVALID"
			}
			before := snapshot(t, root)
			for _, args := range [][]string{run, append(append([]string(nil), run...), "--check"), lifecycleValidate(run)} {
				// Excluding all locked provenance can fail the core first, rather than
				// silently dropping allocations. Both contracts preserve the original tree.
				if kind == "partial" {
					out := invoke(t, base, args...)
					r := result(t, out, 1)
					if len(r.Diagnostics) != 1 || !strings.HasPrefix(r.Diagnostics[0].Code, "E_NAME_LOCK_") {
						t.Fatal(r)
					}
				} else {
					fails(t, invoke(t, base, args...), exit, code)
				}
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("native refusal lost output", kind)
				}
			}
		})
	}
}

func TestLockedOwnerAdoptionNativeSourcePrecedesInputAndReadonlyNoSidecar(t *testing.T) {
	base, cfg, root, _ := lockedAdoptionFixture(t)
	put(t, cfg, []byte(strings.ReplaceAll(string(read(t, cfg)), "catalog.json", "absent.json")))
	put(t, filepath.Join(root, ".openapi-kt-names.json"), []byte("edited"))
	run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	for _, args := range [][]string{run, append(append([]string(nil), run...), "--check"), lifecycleValidate(run)} {
		fails(t, invoke(t, base, args...), 4, "E_NAME_LOCK_MODIFIED")
	}
	base, cfg, root, _ = lockedAdoptionFixture(t)
	if e := os.Remove(filepath.Join(root, ".openapi-kt.lock")); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	dir, _ := os.Stat(root)
	run = []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
	result(t, invoke(t, base, append(run, "--check")...), 3)
	result(t, invoke(t, base, lifecycleValidate(run)...), 0)
	after, _ := os.Stat(root)
	if !reflect.DeepEqual(before, snapshot(t, root)) || !dir.ModTime().Equal(after.ModTime()) {
		t.Fatal("readonly created sidecar/stage or changed directory mtime")
	}
	out := invoke(t, base, "validate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner")
	if out.Code != 0 || !strings.Contains(string(out.Stdout), "manifest v3 -> v4") || !strings.Contains(string(out.Stdout), "descriptor=") || !strings.Contains(string(out.Stdout), "SHA256") {
		t.Fatal("text report", string(out.Stdout), string(out.Stderr))
	}
}
