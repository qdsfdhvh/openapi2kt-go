//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMultiOwnerBusyCheckAndPartialInventory(t *testing.T) {
	root := t.TempDir()
	u := ownersUpdate()
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	lock, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if e := tryLock(lock); e != nil {
		t.Fatal(e)
	}
	defer unlock(lock)
	before := snapshot(t, root)
	u.ActiveOwners = []string{"alpha"}
	u.Generated = map[string]core.GenerationArtifacts{"alpha": ownerArtifacts("AlphaNew")}
	if changed, e := WriteOwners(root, u, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	_, err := WriteOwners(root, u, false)
	errorCode(t, err, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("busy/check changed owners")
	}
	unlock(lock)
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "Beta.kt")); e != nil {
		t.Fatal("partial active inventory removed unrelated beta", e)
	}
}
