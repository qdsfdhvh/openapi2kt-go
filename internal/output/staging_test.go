package output

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStageFilesPreservesOperationSlots(t *testing.T) {
	for _, count := range []int{0, 1, 63, 64, 65, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			stage := t.TempDir()
			operations := make([]operation, count)
			for i := range operations {
				operations[i] = operation{Path: fmt.Sprintf("pkg/Model%d.kt", i), Write: i%3 != 0, Data: []byte(fmt.Sprintf("source%d", i))}
			}
			if err := stageFiles(stage, operations); err != nil {
				t.Fatal(err)
			}
			for i, op := range operations {
				data, err := os.ReadFile(filepath.Join(stage, fmt.Sprintf("new-%d", i)))
				if op.Write {
					if err != nil || !bytes.Equal(data, op.Data) {
						t.Fatal("staging changed an operation's content or index", i, err)
					}
				} else if !os.IsNotExist(err) {
					t.Fatal("removal operation staged a file", i, err)
				}
			}
		})
	}
}

func TestParallelStagingAndCachedParentsRollback(t *testing.T) {
	root := t.TempDir()
	seeded(t, root, map[string][]byte{"a/A.kt": []byte("old"), "a/Stale.kt": []byte("stale")})
	put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
	before := snapshot(t, root)
	next := map[string][]byte{"a/A.kt": []byte("new")}
	for i := range 128 {
		next[fmt.Sprintf("b/nested/B%03d.kt", i)] = []byte(fmt.Sprint(i))
	}
	tx, failure := prepare(root, planned(t, root, next))
	if failure != nil {
		t.Fatal(failure)
	}
	defer tx.cleanup()
	// Remove the last staged write before application starts. No per-operation
	// hook invalidates the directory cache: this exercises the production path
	// through repeated parents, an old-file replacement and a stale removal.
	if err := os.Remove(filepath.Join(tx.stage, fmt.Sprintf("new-%d", len(tx.operations)-1))); err != nil {
		t.Fatal(err)
	}
	if err := tx.apply(nil); err == nil {
		t.Fatal("missing staged manifest did not fail")
	}
	if tx.rollback() {
		t.Fatal("rollback failed")
	}
	tx.cleanup()
	if !reflect.DeepEqual(snapshot(t, root), before) {
		t.Fatal("parallel staging/cached parents changed rollback bytes, mtimes or ownership")
	}
}

func TestStageFilesJoinsWorkersOnFailure(t *testing.T) {
	stage := t.TempDir()
	operations := make([]operation, 257)
	for i := range operations {
		operations[i] = operation{Write: true, Data: []byte(fmt.Sprint(i))}
	}
	// Fail the last operation, so every worker's preceding writes must have
	// completed even when another worker encounters an error.
	if err := os.Mkdir(filepath.Join(stage, "new-256"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := stageFiles(stage, operations); err == nil {
		t.Fatal("staging failure was ignored")
	}
	for i := range 256 {
		data, err := os.ReadFile(filepath.Join(stage, fmt.Sprintf("new-%d", i)))
		if err != nil || !bytes.Equal(data, operations[i].Data) {
			t.Fatal("returned before the remaining workers completed", i, err)
		}
	}
}
