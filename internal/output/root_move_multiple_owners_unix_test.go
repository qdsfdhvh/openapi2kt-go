//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"errors"
	"reflect"
	"testing"
)

func TestRootMoveMultipleOwnersUnixRollbackObstructionRetainsEvidence(t *testing.T) {
	src, dst, s := multipleOwnerMoveFixture(t, true)
	before := snapshot(t, dst)
	path := ""
	for _, candidate := range s.Owners["second"] {
		keep := false
		for _, p := range s.Owners["peer"] {
			keep = keep || p == candidate
		}
		if !keep {
			path = candidate
			break
		}
	}
	if path == "" {
		t.Fatal("fixture lacks second selected exclusive payload")
	}
	_, e := moveOutputSelection(src, dst, s, []string{"second", "alpha"}, false, func(index int, phase string, _ int, _ string) error {
		if phase == "apply" && index == 1 {
			obstructLifecycleRollback(t, src, path)
			return errors.New("synthetic multiple-owner rollback obstruction")
		}
		return nil
	})
	errorCode(t, e, "E_RECOVERY_REQUIRED")
	requireLifecycleBackup(t, src, path, s.Entries[path].Data)
	if !reflect.DeepEqual(before, snapshot(t, dst)) {
		t.Fatal("unobstructed target rollback skipped")
	}
	// The unfinished source journal must remain a subsequent refusal.
	if _, e := ReadRootMoveState(src); e == nil || e.Code != "E_RECOVERY_REQUIRED" {
		t.Fatal("lost recovery evidence", e)
	}
}
