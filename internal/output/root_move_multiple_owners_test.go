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

// Extend the existing mixed/shared merge fixture with a third source owner;
// move alpha and second together while retaining peer. Original noncanonical
// alpha locks, shared target payload attributes and handwritten files stay live.
func multipleOwnerMoveFixture(t testing.TB, owned bool) (string, string, RootMoveState) {
	t.Helper()
	src, dst, _, _ := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
	a := singleGenerated(t, ownerLockRequest(t, "second", core.DataClass, false))
	a.Files = append(a.Files, core.GeneratedFile{Path: "source/Shared.kt", Content: "package source\nobject Shared\n"})
	_, e := WriteNameLockedOwners(src, lockedUpdate([]string{"alpha", "peer", "second"}, map[string]core.GenerationArtifacts{"second": a}, map[string][]byte{}, false), false)
	if e != nil {
		t.Fatal(e)
	}
	if !owned {
		dst = filepath.Join(filepath.Dir(src), "empty-target")
		if err := os.MkdirAll(dst, 0700); err != nil {
			t.Fatal(err)
		}
		// Existing coordination sidecars remain; rollback need not remove a newly
		// created root or writer sidecar.
		put(t, filepath.Join(dst, lockName), nil)
	}
	s, e := ReadRootMoveState(src)
	if e != nil {
		t.Fatal(e)
	}
	s, e = BindRootMoveOwnerTarget(s, dst, src, []string{"second", "alpha"})
	if e != nil {
		t.Fatal(e)
	}
	return src, dst, s
}

func TestRootMoveMultipleOwnersEveryJournalBoundaryRollback(t *testing.T) {
	for _, owned := range []bool{false, true} {
		src, dst, s := multipleOwnerMoveFixture(t, owned)
		a, b := snapshot(t, src), snapshot(t, dst)
		owners := []string{"second", "alpha"}
		counts := map[int]int{}
		prepared := []int{}
		_, e := moveOutputSelection(src, dst, s, owners, false, func(ri int, phase string, op int, stage string) error {
			if phase == "prepare" {
				prepared = append(prepared, ri)
			}
			if phase == "apply" {
				if !reflect.DeepEqual(prepared, []int{0, 1}) {
					t.Fatal("apply before full staging")
				}
				var j journal
				if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); err != nil {
					t.Fatal(err)
				}
				if j.Operations[len(j.Operations)-1].Path != manifestName {
					t.Fatal("manifest not last")
				}
				counts[ri] = len(j.Operations)
				if ri == 1 && op == len(j.Operations)-1 {
					return errors.New("synthetic discover final failure")
				}
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if len(counts) != 2 {
			t.Fatal("missing journals")
		}
		for ri := 0; ri < 2; ri++ {
			for op := 0; op < counts[ri]; op++ {
				_, e := moveOutputSelection(src, dst, s, owners, false, func(index int, phase string, at int, _ string) error {
					if index == ri && phase == "apply" && at == op {
						return errors.New("synthetic multiple-owner apply failure")
					}
					return nil
				})
				errorCode(t, e, "E_OUTPUT_COMMIT")
				if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
					t.Fatal("rollback lost raw state/attributes", owned, ri, op)
				}
			}
			_, e := moveOutputSelection(src, dst, s, owners, false, func(index int, phase string, _ int, _ string) error {
				if index == ri && phase == "prepare" {
					return errors.New("synthetic multiple-owner prepare failure")
				}
				return nil
			})
			errorCode(t, e, "E_OUTPUT_IO")
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("prepare changed roots")
			}
		}
	}
}

func TestRootMoveMultipleOwnersWholeSourceAndTargetCAS(t *testing.T) {
	for _, kind := range []string{"retained-manifest", "retained-attributes", "second-selected-code", "target-manifest"} {
		src, dst, s := multipleOwnerMoveFixture(t, true)
		a, b := snapshot(t, src), snapshot(t, dst)
		_, e := moveOutputSelection(src, dst, s, []string{"second", "alpha"}, false, func(_ int, phase string, _ int, _ string) error {
			if phase != "prelock" {
				return nil
			}
			switch kind {
			case "retained-manifest":
				put(t, filepath.Join(src, manifestName), append(mustRead(t, filepath.Join(src, manifestName)), ' '))
			case "retained-attributes":
				if err := os.Chmod(filepath.Join(src, s.Owners["peer"][0]), 0444); err != nil {
					t.Fatal(err)
				}
			case "second-selected-code":
				put(t, filepath.Join(src, s.Owners["second"][0]), []byte("// intervening synthetic edit"))
			case "target-manifest":
				put(t, filepath.Join(dst, manifestName), append(mustRead(t, filepath.Join(dst, manifestName)), ' '))
			}
			a, b = snapshot(t, src), snapshot(t, dst)
			return nil
		})
		code := "E_OUTPUT_ROOT_MOVE_STALE"
		if kind == "second-selected-code" {
			code = "E_OUTPUT_MODIFIED"
		}
		errorCode(t, e, code)
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("CAS lost intervening state", kind)
		}
	}
}
