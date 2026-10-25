//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveUnrelatedAncestorRebindUnderCompleteLocks(t *testing.T) {
	for _, partial := range []bool{false, true} {
		src, dst, s := moveFixture(t, 4)
		base := filepath.Dir(src)
		first, second := t.TempDir(), t.TempDir()
		alias := filepath.Join(base, "peer-alias")
		if err := os.Symlink(first, alias); err != nil {
			t.Fatal(err)
		}
		s, err := BindRootMoveTarget(s, dst, src)
		if err != nil {
			t.Fatal(err)
		}
		s, err = BindRootMoveUnrelatedRoots(s, []string{filepath.Join(alias, "output")})
		if err != nil {
			t.Fatal(err)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		var owners []string
		if partial {
			owners = []string{"alpha"}
		}
		_, err = moveOutputSelection(src, dst, s, owners, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				if err := os.Remove(alias); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(second, alias); err != nil {
					t.Fatal(err)
				}
			}
			return nil
		})
		errorCode(t, err, "E_OUTPUT_ROOT_MOVE_STALE")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("unrelated binding reclassification changed moved roots")
		}
	}
}
