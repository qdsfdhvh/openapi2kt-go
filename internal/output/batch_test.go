package output

import (
	"encoding/json"
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func batchUpdate(owner string, newer bool) OwnerUpdate {
	files := map[string][]byte{"Stable.kt": []byte("old"), "Removed.kt": []byte("remove"), "Runtime.kt": []byte("runtime")}
	if newer {
		files["Stable.kt"] = []byte("new")
		delete(files, "Removed.kt")
		files["new/Nested.kt"] = []byte("added")
	}
	return OwnerUpdate{ActiveOwners: []string{owner}, Generated: map[string]core.GenerationArtifacts{owner: artifacts(files)}}
}
func batchRoots(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "z")
	if _, e := WriteOwnersBatch(map[string]OwnerUpdate{b: batchUpdate("beta", false), a: batchUpdate("alpha", false)}, false); e != nil {
		t.Fatal(e)
	}
	for _, root := range []string{a, b} {
		put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
	}
	return a, b
}
func TestBatchEveryRootAndOperationRollback(t *testing.T) {
	for targetRoot := range 2 {
		for targetOp := range 4 {
			for _, stagedFail := range []bool{false, true} {
				a, b := batchRoots(t)
				oldA, oldB := snapshot(t, a), snapshot(t, b)
				_, e := writeOwnersBatch(map[string]OwnerUpdate{b: batchUpdate("beta", true), a: batchUpdate("alpha", true)}, false, func(root int, phase string, index int, stage string) error {
					if phase != "apply" || root != targetRoot || index != targetOp {
						return nil
					}
					if stagedFail {
						var j journal
						if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
							t.Fatal(e)
						}
						if p := j.Operations[index].Staged; p != nil {
							return os.Remove(filepath.Join(stage, *p))
						}
					}
					return errors.New("injected batch failure")
				})
				errorCode(t, e, "E_OUTPUT_COMMIT")
				if !reflect.DeepEqual(oldA, snapshot(t, a)) || !reflect.DeepEqual(oldB, snapshot(t, b)) {
					t.Fatal("batch rollback changed bytes/mtime/mode", targetRoot, targetOp, stagedFail)
				}
			}
		}
	}
}
func TestBatchPrepareAllBeforeApplyAndNoop(t *testing.T) {
	a, b := batchRoots(t)
	updates := map[string]OwnerUpdate{a: batchUpdate("alpha", true), b: batchUpdate("beta", true)}
	beforeA, beforeB := snapshot(t, a), snapshot(t, b)
	for failRoot := range 2 {
		_, e := writeOwnersBatch(updates, false, func(root int, phase string, index int, stage string) error {
			if phase == "prepare" {
				if string(mustRead(t, filepath.Join(a, "Stable.kt"))) != "old" {
					t.Fatal("first root applied before second prepared")
				}
				if root == failRoot {
					return errors.New("prepare failure")
				}
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_IO")
		if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
			t.Fatal("preparation changed outputs")
		}
	}
	applySeen := false
	changed, e := writeOwnersBatch(updates, false, func(root int, phase string, index int, stage string) error {
		if phase == "apply" && !applySeen {
			applySeen = true
			for _, root := range []string{a, b} {
				entries, e := os.ReadDir(root)
				if e != nil {
					t.Fatal(e)
				}
				stages := 0
				for _, entry := range entries {
					if len(entry.Name()) >= len(stagePrefix) && entry.Name()[:len(stagePrefix)] == stagePrefix {
						stages++
					}
				}
				if stages != 1 {
					t.Fatal("not all roots staged", stages)
				}
			}
		}
		return nil
	})
	if e != nil || !changed || !applySeen {
		t.Fatal(changed, e)
	}
	beforeA, beforeB = snapshot(t, a), snapshot(t, b)
	for _, check := range []bool{true, false} {
		if c, e := WriteOwnersBatch(updates, check); e != nil || c {
			t.Fatal(c, e)
		}
	}
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("batch noop/check changed output")
	}
}
func TestBatchReplanRetainsInterveningOwner(t *testing.T) {
	a, b := batchRoots(t)
	updates := map[string]OwnerUpdate{a: batchUpdate("alpha", true), b: batchUpdate("beta", true)}
	called := false
	_, e := writeOwnersBatch(updates, false, func(root int, phase string, index int, stage string) error {
		if phase == "prelock" {
			called = true
			other := OwnerUpdate{ActiveOwners: []string{"beta", "gamma"}, Generated: map[string]core.GenerationArtifacts{"gamma": artifacts(map[string][]byte{"Gamma.kt": []byte("gamma")})}}
			if _, e := WriteOwners(b, other, false); e != nil {
				t.Fatal(e)
			}
		}
		return nil
	})
	if e != nil || !called {
		t.Fatal(e)
	}
	if string(mustRead(t, filepath.Join(b, "Gamma.kt"))) != "gamma" {
		t.Fatal("lost intervening owner")
	}
}
func TestBatchOverlappingRootAndCheckNoIO(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "z")
	u := batchUpdate("alpha", true)
	if c, e := WriteOwnersBatch(map[string]OwnerUpdate{a: u, b: u}, true); e != nil || !c {
		t.Fatal(c, e)
	}
	entries, e := os.ReadDir(base)
	if e != nil || len(entries) != 0 {
		t.Fatal("check created directory", e)
	}
	for _, roots := range []map[string]OwnerUpdate{{a: u, filepath.Join(a, "nested"): u}, {a: u, filepath.Join(base, "A"): u}} {
		_, e := WriteOwnersBatch(roots, false)
		errorCode(t, e, "E_OUTPUT_ROOTS")
	}
	entries, e = os.ReadDir(base)
	if e != nil || len(entries) != 0 {
		t.Fatal("root refusal created directory", e)
	}
}
