package output

import (
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAdoptionBatchPeerFailureRestoresBothVersions(t *testing.T) {
	for targetOp := range 4 {
		base := t.TempDir()
		a, b := filepath.Join(base, "a"), filepath.Join(base, "z")
		files := ownerArtifacts("Alpha")
		if _, e := Write(a, files, false); e != nil {
			t.Fatal(e)
		}
		if _, e := WriteOwners(b, batchUpdate("beta", false), false); e != nil {
			t.Fatal(e)
		}
		beforeA, beforeB := snapshot(t, a), snapshot(t, b)
		u := OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": files}, AdoptLegacy: "alpha"}
		_, e := writeOwnersBatch(map[string]OwnerUpdate{a: u, b: batchUpdate("beta", true)}, false, func(root int, phase string, index int, stage string) error {
			if root == 1 && phase == "apply" && index == targetOp {
				return errors.New("later peer failure")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
			t.Fatal("adoption+peer rollback lost legacy ownership", targetOp)
		}
	}
}
func TestAdoptionFailedRollbackRetainsOriginalV1Manifest(t *testing.T) {
	root, u := legacyOwnerRoot(t)
	oldManifest := mustRead(t, filepath.Join(root, manifestName))
	p, e := makeOwnersPlan(root, u)
	if e != nil {
		t.Fatal(e)
	}
	tx, err := prepare(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if e := os.Remove(filepath.Join(tx.stage, "new-0")); e != nil {
		t.Fatal(e)
	}
	if tx.apply(nil) == nil {
		t.Fatal("missing staged manifest unexpectedly applied")
	}
	if e := os.Mkdir(filepath.Join(root, manifestName), 0700); e != nil {
		t.Fatal(e)
	}
	if !tx.rollback() {
		t.Fatal("obstructed manifest restore unexpectedly succeeded")
	}
	tx.cleanup()
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	found := false
	for _, entry := range entries {
		if len(entry.Name()) >= len(stagePrefix) && entry.Name()[:len(stagePrefix)] == stagePrefix {
			found = true
			if !reflect.DeepEqual(oldManifest, mustRead(t, filepath.Join(root, entry.Name(), "old-0"))) {
				t.Fatal("lost original legacy manifest")
			}
		}
	}
	if !found {
		t.Fatal("lost recovery stage")
	}
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		_, err := WriteOwners(root, u, check)
		errorCode(t, err, "E_RECOVERY_REQUIRED")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("recovery refusal changed output")
	}
}
