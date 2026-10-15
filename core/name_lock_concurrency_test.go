package core_test

import (
	"bytes"
	"openapi-kt/core"
	"reflect"
	"sync"
	"testing"
)

func TestNameLockConcurrentReadonlyAndDepth(t *testing.T) {
	req := request(nameLockInput(t), "Dto")
	req.Options.HoistSharedShapes = true
	req.Options.LockHoistedNames = true
	req.Options.Validation = true
	initial, e := core.GenerateWithDiagnostics(req)
	if e != nil {
		t.Fatal(e)
	}
	req.PreviousNameLock = initial.Artifacts.NameLock
	doc, prior := append([]byte(nil), req.Document...), append([]byte(nil), req.PreviousNameLock...)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(initial, r) {
				t.Error("concurrent nondeterminism", e)
			}
		})
	}
	wg.Wait()
	if !bytes.Equal(doc, req.Document) || !bytes.Equal(prior, req.PreviousNameLock) {
		t.Fatal("core modified caller bytes")
	}
	for _, n := range []int{49, 50} {
		r := request(rootObjectChain(n, ""), "")
		r.Options.HoistSharedShapes = true
		base, e := core.GenerateWithDiagnostics(r)
		r.Options.LockHoistedNames = true
		locked, f := core.GenerateWithDiagnostics(r)
		if e == nil || f == nil || !reflect.DeepEqual(e, f) || !reflect.DeepEqual(base, locked) {
			t.Fatal("depth guard lost", n, e, f)
		}
	}
}
