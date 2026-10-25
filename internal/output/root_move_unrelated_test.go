package output

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveUnrelatedInventoryBoundaries(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	bound, err := BindRootMoveTarget(s, dst, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{src, dst, filepath.Join(src, "child"), filepath.Join(dst, "child"), filepath.Dir(src)} {
		a, b := snapshot(t, src), snapshot(t, dst)
		if _, err := BindRootMoveUnrelatedRoots(bound, []string{root}); err == nil {
			t.Fatal("related config owner classified as unrelated", root)
		}
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("binding classification wrote output")
		}
	}
	peer := filepath.Join(filepath.Dir(src), "peer", "absent")
	observed, err := BindRootMoveUnrelatedRoots(bound, []string{peer})
	if err != nil || len(observed.bindings) != len(bound.bindings)+1 {
		t.Fatal("lost unrelated binding", err)
	}
	if _, err := MoveOutputRoot(src, dst, observed, true); err != nil {
		t.Fatal(err)
	}
}
