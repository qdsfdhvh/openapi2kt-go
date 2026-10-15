package output

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBatchFailedRollbackKeepsEvidenceAndRestoresOtherRoot(t *testing.T) {
	a, b := batchRoots(t)
	oldA := snapshot(t, a)
	updates := map[string]OwnerUpdate{a: batchUpdate("alpha", true), b: batchUpdate("beta", true)}
	_, e := writeOwnersBatch(updates, false, func(root int, phase string, index int, stage string) error {
		if root == 1 && phase == "apply" && index == 3 {
			path := filepath.Join(b, "Stable.kt")
			if e := os.Remove(path); e != nil {
				t.Fatal(e)
			}
			if e := os.Mkdir(path, 0700); e != nil {
				t.Fatal(e)
			}
			return errors.New("rollback obstruction")
		}
		return nil
	})
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	if !reflect.DeepEqual(oldA, snapshot(t, a)) {
		t.Fatal("other root not restored after failed peer rollback")
	}
	entries, err := os.ReadDir(b)
	if err != nil {
		t.Fatal(err)
	}
	stages := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagePrefix) {
			stages++
			stage := filepath.Join(b, entry.Name())
			if string(mustRead(t, filepath.Join(stage, "old-0"))) != "old" {
				t.Fatal("lost old stable backup")
			}
			if len(mustRead(t, filepath.Join(stage, "transaction.json"))) == 0 {
				t.Fatal("lost journal")
			}
		}
	}
	if stages != 1 {
		t.Fatal(stages)
	}
	beforeA, beforeB := snapshot(t, a), snapshot(t, b)
	for _, check := range []bool{true, false} {
		_, e := WriteOwnersBatch(updates, check)
		errorCode(t, e, "E_RECOVERY_REQUIRED")
	}
	if !reflect.DeepEqual(beforeA, snapshot(t, a)) || !reflect.DeepEqual(beforeB, snapshot(t, b)) {
		t.Fatal("recovery refusal changed outputs")
	}
}
