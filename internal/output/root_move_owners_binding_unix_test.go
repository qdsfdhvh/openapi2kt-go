//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveOwnersTargetAndDeclaredRebindUnderCompleteLocks(t *testing.T) {
	for _, kind := range []string{"target", "declared"} {
		src, dst, s := moveFixture(t, 4)
		base := filepath.Dir(src)
		alternate := t.TempDir()
		alias := filepath.Join(base, "alias")
		if e := os.Symlink(base, alias); e != nil {
			t.Fatal(e)
		}
		target, declared := filepath.Join(alias, filepath.Base(dst)), dst
		if kind == "declared" {
			target, declared = dst, filepath.Join(alias, filepath.Base(dst))
		}
		bound, e := BindRootMoveOwnerTarget(s, target, declared, []string{"alpha"})
		if e != nil {
			t.Fatal(e)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		_, e = moveOutputSelection(src, target, bound, []string{"alpha"}, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				if e := os.Remove(alias); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(alternate, alias); e != nil {
					t.Fatal(e)
				}
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_ROOT_MOVE_STALE")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("locked binding CAS lost output")
		}
	}
}
