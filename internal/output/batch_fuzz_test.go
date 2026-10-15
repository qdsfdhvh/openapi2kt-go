package output

import (
	"errors"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func FuzzBatchRollback(f *testing.F) {
	for _, seed := range [][]byte{{0, 0}, {0, 3}, {1, 0}, {1, 3, 'x'}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) < 2 || len(b) > 256 {
			t.Skip()
		}
		a, z := batchRoots(t)
		beforeA, beforeZ := snapshot(t, a), snapshot(t, z)
		makeUpdate := func(owner string) OwnerUpdate {
			return OwnerUpdate{ActiveOwners: []string{owner}, Generated: map[string]core.GenerationArtifacts{owner: artifacts(map[string][]byte{"Stable.kt": append([]byte("changed:"), b...), "new/Nested.kt": []byte("new"), "Runtime.kt": []byte("runtime")})}}
		}
		hit := false
		_, e := writeOwnersBatch(map[string]OwnerUpdate{z: makeUpdate("beta"), a: makeUpdate("alpha")}, false, func(root int, phase string, index int, stage string) error {
			if phase == "apply" && root == int(b[0]%2) && index == int(b[1]%4) {
				hit = true
				return errors.New("fuzzed batch failure")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !hit || !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeZ, snapshot(t, z)) {
			t.Fatal("cross-root rollback model mismatch", filepath.Base(a), hit)
		}
	})
}
