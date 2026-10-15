//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSingleNameLockBusyWriteReadonlyCheckAndLockedCAS(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	a := singleGenerated(t, req)
	if _, e := WriteNameLocked(root, a, nil, false); e != nil {
		t.Fatal(e)
	}
	holder, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer holder.Close()
	if e := tryLock(holder); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	if state, e := ReadNameLock(root); e != nil || string(state) != string(a.NameLock) {
		t.Fatal(e)
	}
	if changed, e := WriteNameLocked(root, a, a.NameLock, true); e != nil || changed {
		t.Fatal(e)
	}
	_, failure := WriteNameLocked(root, a, a.NameLock, false)
	errorCode(t, failure, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("busy/check mutated state")
	}
	unlock(holder)
	calls := 0
	_, failure = writePlanned(root, false, func(root string) (*plan, *diagnostic.Diagnostic) {
		calls++
		if calls == 2 {
			competing, e := os.OpenFile(filepath.Join(root, lockName), os.O_RDWR, 0600)
			if e != nil {
				t.Fatal(e)
			}
			defer competing.Close()
			if e := tryLock(competing); e == nil {
				unlock(competing)
				t.Fatal("second CAS plan not under flock")
			}
		}
		return makeNameLockedPlan(root, a, a.NameLock)
	})
	if failure != nil || calls != 2 {
		t.Fatal(calls, failure)
	}
}
