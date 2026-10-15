//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadNameLockFilesDoesNotAcquireBusyWriterLock(t *testing.T) {
	for _, named := range []bool{false, true} {
		root := t.TempDir()
		a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
		seedLifecycleTest(t, root, named, a)
		owner := ""
		if named {
			owner = "alpha"
		}
		before := snapshot(t, root)
		holder, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		if e := tryLock(holder); e != nil {
			_ = holder.Close()
			t.Fatal(e)
		}
		files, failure := ReadNameLockFiles(root, owner, a.NameLock)
		unlock(holder)
		if e := holder.Close(); e != nil {
			t.Fatal(e)
		}
		if failure != nil || !reflect.DeepEqual(files, sortedNameLockTestFiles(a.Files)) {
			t.Fatal("readonly content lookup attempted writer acquisition", files, failure)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("busy readonly lookup changed bytes/modes/mtimes")
		}
	}
}
