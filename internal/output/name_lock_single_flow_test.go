package output

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func addedSinglePeer(t testing.TB, req core.GenerationRequest) core.GenerationRequest {
	t.Helper()
	var d, paths, alpha, post map[string]json.RawMessage
	if e := json.Unmarshal(req.Document, &d); e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal(d["paths"], &paths)
	_ = json.Unmarshal(paths["/alpha"], &alpha)
	_ = json.Unmarshal(alpha["post"], &post)
	post["operationId"] = json.RawMessage(`"aardvark"`)
	alpha["post"], _ = json.Marshal(post)
	paths["/aardvark"], _ = json.Marshal(alpha)
	d["paths"], _ = json.Marshal(paths)
	req.Document, _ = json.Marshal(d)
	return req
}
func TestSingleNameLockReadGenerateCheckCommitReplay(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		root := filepath.Join(t.TempDir(), "absent")
		before, e := ReadNameLock(root)
		if e != nil || before != nil {
			t.Fatal(e)
		}
		req := singleLockRequest(t, style, true)
		req.PreviousNameLock = before
		a := singleGenerated(t, req)
		if changed, e := WriteNameLocked(root, a, before, true); e != nil || !changed {
			t.Fatal(e)
		}
		if _, e := os.Stat(root); !os.IsNotExist(e) {
			t.Fatal("check/read created root")
		}
		if changed, e := WriteNameLocked(root, a, before, false); e != nil || !changed {
			t.Fatal(e)
		}
		before, e = ReadNameLock(root)
		if e != nil || !bytes.Equal(before, a.NameLock) {
			t.Fatal(e)
		}
		stable := snapshot(t, root)
		for _, check := range []bool{true, false} {
			if changed, e := WriteNameLocked(root, a, before, check); e != nil || changed {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(stable, snapshot(t, root)) {
				t.Fatal("noop changed state/tree")
			}
		}
		req = addedSinglePeer(t, req)
		req.PreviousNameLock = before
		next := singleGenerated(t, req)
		if bytes.Equal(next.NameLock, before) {
			t.Fatal("new occurrence missing")
		}
		if changed, e := WriteNameLocked(root, next, before, true); e != nil || !changed {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(stable, snapshot(t, root)) {
			t.Fatal("difference check mutated output")
		}
		if changed, e := WriteNameLocked(root, next, before, false); e != nil || !changed {
			t.Fatal(e)
		}
		after, e := ReadNameLock(root)
		if e != nil || !bytes.Equal(after, next.NameLock) {
			t.Fatal(e)
		}
		req.PreviousNameLock = after
		replay := singleGenerated(t, req)
		if !reflect.DeepEqual(replay, next) {
			t.Fatal("read replay differs")
		}
		for _, f := range next.Files {
			if !bytes.Equal(mustRead(t, filepath.Join(root, f.Path)), []byte(f.Content)) {
				t.Fatal("committed Kotlin differs from core")
			}
		}
		final := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLocked(root, next, before, check)
			errorCode(t, e, "E_NAME_LOCK_STALE")
			if !reflect.DeepEqual(final, snapshot(t, root)) {
				t.Fatal("stale generation mutated output")
			}
		}
	}
}
func TestSingleNameLockVersionOneMetadataOnlyMigration(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, false)
	plain := singleGenerated(t, req)
	if _, e := Write(root, plain, false); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	req.Options.LockHoistedNames = true
	locked := singleGenerated(t, req)
	if !reflect.DeepEqual(plain.Files, locked.Files) {
		t.Fatal("migration changed Kotlin generation")
	}
	if changed, e := WriteNameLocked(root, locked, nil, true); e != nil || !changed {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("migration check wrote")
	}
	if changed, e := WriteNameLocked(root, locked, nil, false); e != nil || !changed {
		t.Fatal(e)
	}
	after := snapshot(t, root)
	for _, f := range plain.Files {
		if !reflect.DeepEqual(before[f.Path], after[f.Path]) {
			t.Fatal("metadata-only migration rewrote Kotlin", f.Path)
		}
	}
	for _, check := range []bool{true, false} {
		_, e := Write(root, plain, check)
		errorCode(t, e, "E_MANIFEST_VERSION")
		_, e = WriteOwners(root, OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": plain}}, check)
		errorCode(t, e, "E_MANIFEST_VERSION")
	}
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("default/named downgrade mutated v3")
	}
}
func TestSingleNameLockCASRechecksUnderWriterLock(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	original := singleGenerated(t, req)
	if _, e := WriteNameLocked(root, original, nil, false); e != nil {
		t.Fatal(e)
	}
	req = addedSinglePeer(t, req)
	req.PreviousNameLock = original.NameLock
	competing := singleGenerated(t, req)
	calls := 0
	var afterCompeting map[string]snapshotEntry
	_, e := writePlanned(root, false, func(root string) (*plan, *diagnostic.Diagnostic) {
		calls++
		if calls == 2 {
			// A cooperative update between snapshot/preflight and acquisition is
			// represented by its complete committed tree. This hook is test-only.
			p, e := makeNameLockedPlan(root, competing, original.NameLock)
			if e != nil {
				t.Fatal(e)
			}
			if e := commit(root, p, nil); e != nil {
				t.Fatal(e)
			}
			afterCompeting = snapshot(t, root)
		}
		return makeNameLockedPlan(root, original, original.NameLock)
	})
	errorCode(t, e, "E_NAME_LOCK_STALE")
	if calls != 2 || !reflect.DeepEqual(afterCompeting, snapshot(t, root)) {
		t.Fatal("under-lock CAS missing/changed competitor output")
	}
}
