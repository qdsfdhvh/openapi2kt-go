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

func TestRootMoveNativeSelectedOwnerAndUnchangedPeer(t *testing.T) {
	for _, locked := range []bool{false, true} {
		for _, style := range []string{"data_class", "plain_class"} {
			base := t.TempDir()
			names := []string{"alpha", "beta"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, locked, false, false)
			text := string(read(t, cfg))
			text = strings.Replace(text, "[defaults]", "[defaults]\nmodel_style='"+style+"'", 1)
			put(t, cfg, []byte(text))
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, base, args...), 0)
			src, dst := filepath.Join(base, "generated"), filepath.Join(base, "new")
			before := snapshot(t, src)
			var original struct{ Owners map[string][]string }
			if err := json.Unmarshal(read(t, filepath.Join(src, ".openapi-kt-manifest.json")), &original); err != nil {
				t.Fatal(err)
			}
			run := append(append([]string(nil), args...), "--move-output-from", src, "--output", dst, "--spec=alpha")
			checked := invoke(t, base, append(append([]string(nil), run...), "--check")...)
			result(t, checked, 3)
			validated := invoke(t, base, lifecycleValidate(run)...)
			result(t, validated, 0)
			_, targetError := os.Lstat(dst)
			if !reflect.DeepEqual(before, snapshot(t, src)) || !os.IsNotExist(targetError) {
				t.Fatal("partial check mutated output")
			}
			committed := invoke(t, base, run...)
			result(t, committed, 0)
			for _, out := range []invocation{checked, validated, committed} {
				var raw struct {
					Migration struct {
						Moved    []string `json:"moved_owners"`
						Retained []string `json:"retained_owners"`
						Files    []struct{ Before string }
					} `json:"output_root_migration"`
				}
				if err := json.Unmarshal(out.Stdout, &raw); err != nil || !reflect.DeepEqual(raw.Migration.Moved, []string{"alpha"}) || !reflect.DeepEqual(raw.Migration.Retained, []string{"beta"}) || len(raw.Migration.Files) != len(original.Owners["alpha"]) || bytes.Contains(out.Stdout, []byte(base)) {
					t.Fatal("missing selected/retained inventory", err, string(out.Stdout))
				}
			}
			afterSrc, afterDst := snapshot(t, src), snapshot(t, dst)
			for _, path := range original.Owners["alpha"] {
				if !reflect.DeepEqual(before[path], afterDst[path]) {
					t.Fatal("moved payload changed", path)
				}
			}
			for _, path := range original.Owners["beta"] {
				if !reflect.DeepEqual(before[path], afterSrc[path]) {
					t.Fatal("retained peer changed", path)
				}
			}
			if locked {
				if !reflect.DeepEqual(before[namedLockPath("alpha")], afterDst[namedLockPath("alpha")]) || !reflect.DeepEqual(before[namedLockPath("beta")], afterSrc[namedLockPath("beta")]) {
					t.Fatal("changed owned locks")
				}
			}
			// Adopt the new configured root explicitly, then ordinary generation/check
			// must validate the split inventories without finding the old root.
			text = strings.Replace(text, "name=\"alpha\"", "name=\"alpha\"\noutput_dir='new'", 1)
			put(t, cfg, []byte(text))
			result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 0)
			result(t, invoke(t, base, args...), 0)
			if !reflect.DeepEqual(afterSrc, snapshot(t, src)) || !reflect.DeepEqual(afterDst, snapshot(t, dst)) {
				t.Fatal("ordinary generation rewrote split output")
			}
		}
	}
}
