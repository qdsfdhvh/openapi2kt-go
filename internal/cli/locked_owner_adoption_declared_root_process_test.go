package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Same-root adoption uses the selected effective root. Its ordinary CLI output
// override must not weaken root move's separate, pre-override declaration guard.
func TestLockedOwnerAdoptionNativeExplicitRootOverrideKeepsMoveDeclarationGuard(t *testing.T) {
	for _, kind := range []string{"omitted", "empty-default", "empty-spec-overrides-default"} {
		t.Run(kind, func(t *testing.T) {
			base, cfg, root, _ := lockedAdoptionFixture(t)
			text := string(read(t, cfg))
			switch kind {
			case "omitted":
				text = strings.ReplaceAll(text, "output_dir='generated',", "")
			case "empty-default":
				text = strings.ReplaceAll(text, "output_dir='generated'", "output_dir=''")
			case "empty-spec-overrides-default":
				text = strings.ReplaceAll(text, "input='catalog.json'", "input='catalog.json',output_dir=''")
			}
			put(t, cfg, []byte(text))
			before := snapshot(t, root)
			directory, e := os.Stat(root)
			if e != nil {
				t.Fatal(e)
			}
			adopt := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--output", root, "--diagnostic-format=json"}
			result(t, invoke(t, base, append(append([]string(nil), adopt...), "--check")...), 3)
			result(t, invoke(t, base, lifecycleValidate(adopt)...), 0)
			afterReadonly, e := os.Stat(root)
			if e != nil || !reflect.DeepEqual(before, snapshot(t, root)) || !directory.ModTime().Equal(afterReadonly.ModTime()) {
				t.Fatal("same-root override preflight changed output", e)
			}
			result(t, invoke(t, base, adopt...), 0)
			after := snapshot(t, root)
			for path, file := range before {
				if strings.HasSuffix(path, ".kt") && !reflect.DeepEqual(file, after[path]) {
					t.Fatal("adoption changed Kotlin bytes/modes/mtimes", path)
				}
			}
			if before[".openapi-kt-names.json"].Bytes != after[namedLockPath("catalog")].Bytes {
				t.Fatal("adoption changed raw lock payload")
			}
			result(t, invoke(t, base, "generate", "--config", cfg, "--spec=catalog", "--output", root, "--check", "--diagnostic-format=json"), 0)

			// Use the now-valid complete v4 named source so the refusal specifically
			// proves root move rejects the declaration before state/input IO.
			target := filepath.Join(base, "missing", "target")
			move := []string{"generate", "--config", cfg, "--move-output-from", root, "--output", target, "--metrics", "--diagnostic-format=json"}
			for _, mode := range []string{"check", "validate", "apply"} {
				requireRootMoveInventoryRefusalBeforeIO(t, invoke(t, base, rootMoveInvocation(move, mode)...))
				if !reflect.DeepEqual(after, snapshot(t, root)) {
					t.Fatal("root move declaration refusal changed adopted source", mode)
				}
				if _, e := os.Stat(filepath.Dir(target)); !os.IsNotExist(e) {
					t.Fatal("root move declaration refusal created target parent", e)
				}
			}
		})
	}
}
