//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveOwnersUnixBusyCompleteSortedLocksAndStages(t *testing.T) {
	for _, busySource := range []bool{false, true} {
		src, dst, s := moveFixture(t, 4)
		busyRoot := dst
		if busySource {
			busyRoot = src
		}
		f, e := os.OpenFile(filepath.Join(busyRoot, lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := tryLock(f); e != nil {
			t.Fatal(e)
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		if changed, e := MoveOutputOwners(src, dst, s, []string{"alpha"}, true); e != nil || !changed {
			t.Fatal(changed, e)
		}
		_, err := MoveOutputOwners(src, dst, s, []string{"alpha"}, false)
		errorCode(t, err, "E_OUTPUT_BUSY")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("busy/check changed output")
		}
		unlock(f)
		_ = f.Close()
		prepared := []int{}
		_, err = moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(index int, phase string, op int, stage string) error {
			if phase == "prepare" {
				prepared = append(prepared, index)
			}
			if phase == "apply" && op == 0 {
				if !reflect.DeepEqual(prepared, []int{0, 1}) {
					t.Fatal("apply before all preparation", prepared)
				}
				for _, root := range []string{src, dst} {
					f, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
					if e != nil {
						t.Fatal(e)
					}
					e = tryLock(f)
					if e == nil {
						unlock(f)
					}
					_ = f.Close()
					if e == nil {
						t.Fatal("complete locks missing")
					}
					entries, e := os.ReadDir(root)
					if e != nil {
						t.Fatal(e)
					}
					found := false
					for _, entry := range entries {
						if len(entry.Name()) > len(stagePrefix) && entry.Name()[:len(stagePrefix)] == stagePrefix {
							found = true
						}
					}
					if !found {
						t.Fatal("root not staged")
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal("released lock leaked", err)
		}
	}
}
