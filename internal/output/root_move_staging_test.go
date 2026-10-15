package output

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestRootMoveStagingAttributeFailuresBeforeApply(t *testing.T) {
	for _, phase := range []string{"chmod", "chtimes"} {
		src, dst, s := moveFixture(t, 3)
		a, b := snapshot(t, src), snapshot(t, dst)
		p := &plan{Changed: true, Writes: map[string][]byte{}, Attributes: map[string]fileAttributes{}, ManifestBytes: s.Manifest}
		for path, entry := range s.Entries {
			p.Attributes[path] = entry.Attributes
			if path != manifestName {
				p.Writes[path] = entry.Data
			}
		}
		reached := false
		_, e := prepareWithAttributes(dst, p, func(at, path string) error {
			if at == phase {
				reached = true
				return os.Remove(path)
			} // Real syscall failure after the staged file disappears.
			return nil
		})
		errorCode(t, e, "E_OUTPUT_IO")
		if !reached || !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("attribute preparation mutated old output")
		}
	}
}

func TestRootMoveFailureAfterTargetManifestStagedFileLost(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	a, b := snapshot(t, src), snapshot(t, dst)
	_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, op int, stage string) error {
		if index == 1 && phase == "apply" && op == len(s.Entries)-1 {
			return os.Remove(filepath.Join(stage, "new-"+strconv.Itoa(op)))
		}
		return nil
	})
	errorCode(t, e, "E_OUTPUT_COMMIT")
	if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
		t.Fatal("lost raw manifest apply restored incomplete roots")
	}
}
