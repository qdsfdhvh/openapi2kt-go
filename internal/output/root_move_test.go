package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/core"
)

func moveFixture(t testing.TB, version int) (string, string, RootMoveState) {
	t.Helper()
	base := t.TempDir()
	source, target := filepath.Join(base, "a-source"), filepath.Join(base, "z-target")
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, version >= 3))
	var e *core.Diagnostic
	switch version {
	case 1:
		_, e = Write(source, a, false)
	case 3:
		_, e = WriteNameLocked(source, a, nil, false)
	default:
		u := lockedUpdate([]string{"alpha", "peer"}, map[string]core.GenerationArtifacts{"alpha": a, "peer": a}, nil, true)
		if version == 4 {
			u.PreviousNameLocks = map[string][]byte{"alpha": nil, "peer": nil}
		}
		_, e = WriteNameLockedOwners(source, u, false)
	}
	if e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(target, 0700); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(target, lockName), []byte{})
	put(t, filepath.Join(source, "Notes.kt"), []byte("// source unowned"))
	put(t, filepath.Join(target, "Notes.kt"), []byte("// target unowned"))
	// Noncanonical valid serialization must migrate verbatim, not be adopted.
	put(t, filepath.Join(source, manifestName), append(mustRead(t, filepath.Join(source, manifestName)), '\n'))
	stamp := time.Unix(1200000000, 0)
	attributeIndex := 0
	for path, entry := range snapshot(t, source) {
		if entry.Directory || path == lockName {
			continue
		}
		p := filepath.Join(source, path)
		attributeIndex++
		mode := os.FileMode(0600)
		if attributeIndex%2 == 0 {
			mode = 0640
		}
		modified := stamp.Add(time.Duration(attributeIndex) * time.Second)
		if e := os.Chmod(p, mode); e != nil {
			t.Fatal(e)
		}
		if e := os.Chtimes(p, modified, modified); e != nil {
			t.Fatal(e)
		}
	}
	s, e := ReadRootMoveState(source)
	if e != nil {
		t.Fatal(e)
	}
	return source, target, s
}

func TestRootMoveAllVersionsByteAttributesMembershipAndReadonly(t *testing.T) {
	for _, version := range []int{1, 2, 3, 4} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			src, dst, s := moveFixture(t, version)
			oldSrc, oldDst := snapshot(t, src), snapshot(t, dst)
			for _, check := range []bool{true, false} {
				changed, e := MoveOutputRoot(src, dst, s, check)
				if e != nil || !changed {
					t.Fatal(changed, e)
				}
				if check {
					if !reflect.DeepEqual(oldSrc, snapshot(t, src)) || !reflect.DeepEqual(oldDst, snapshot(t, dst)) {
						t.Fatal("check changed output")
					}
					continue
				}
				after := snapshot(t, dst)
				for path := range s.Entries {
					key := filepath.FromSlash(path)
					if !reflect.DeepEqual(oldSrc[key], after[key]) {
						t.Fatal("copy changed bytes/mode/mtime", path)
					}
					if _, e := os.Lstat(filepath.Join(src, key)); !os.IsNotExist(e) {
						t.Fatal("owned source survived", path, e)
					}
				}
				if !reflect.DeepEqual(oldSrc["Notes.kt"], snapshot(t, src)["Notes.kt"]) || !reflect.DeepEqual(oldDst["Notes.kt"], after["Notes.kt"]) {
					t.Fatal("unowned file changed")
				}
				next, e := ReadRootMoveState(dst)
				expected := s
				expected.root = next.root
				if e != nil || !reflect.DeepEqual(next, expected) {
					t.Fatal("ownership/allocations changed", e)
				}
				for _, readOnly := range []bool{true, false} {
					changed, e := MoveOutputRoot(dst, dst, next, readOnly)
					if e != nil || changed {
						t.Fatal("identity", changed, e)
					}
					if !reflect.DeepEqual(after, snapshot(t, dst)) {
						t.Fatal("identity wrote")
					}
				}
			}
		})
	}
}

func TestRootMoveRefusalsPreserveBothTrees(t *testing.T) {
	for _, kind := range []string{"missing", "modified", "manifest", "lock", "foreign-source", "foreign-target", "occupied", "collision", "recovery-source", "recovery-target", "stale-manifest", "stale-payload"} {
		t.Run(kind, func(t *testing.T) {
			src, dst, s := moveFixture(t, 4)
			path := s.Files[0].Path
			code := "E_OUTPUT_ROOT_MOVE_STALE"
			switch kind {
			case "missing":
				if e := os.Remove(filepath.Join(src, path)); e != nil {
					t.Fatal(e)
				}
				code = "E_OUTPUT_MISSING"
			case "modified":
				put(t, filepath.Join(src, path), []byte("edited"))
				code = "E_OUTPUT_MODIFIED"
			case "manifest":
				put(t, filepath.Join(src, manifestName), []byte(`{"version":4,"files":false}`))
				code = "E_MANIFEST_INVALID"
			case "lock":
				put(t, filepath.Join(src, ownerNameLockFile("peer")), []byte("edited"))
				code = "E_NAME_LOCK_MODIFIED"
			case "foreign-source":
				put(t, filepath.Join(src, nameLockFile), []byte("unowned"))
				code = "E_OUTPUT_ROOT_MOVE_INVALID"
			case "foreign-target":
				put(t, filepath.Join(dst, ownerNameLockFile("foreign")), []byte("unowned"))
				code = "E_OUTPUT_ROOT_MOVE_INVALID"
			case "occupied":
				put(t, filepath.Join(dst, manifestName), s.Manifest)
				code = "E_OUTPUT_ROOT_MOVE_OCCUPIED"
			case "collision":
				if e := os.MkdirAll(filepath.Dir(filepath.Join(dst, path)), 0700); e != nil {
					t.Fatal(e)
				}
				put(t, filepath.Join(dst, path), []byte(s.Files[0].Content))
				code = "E_OUTPUT_COLLISION"
			case "recovery-source", "recovery-target":
				root := src
				if kind == "recovery-target" {
					root = dst
				}
				if e := os.Mkdir(filepath.Join(root, stagePrefix+"legacy"), 0700); e != nil {
					t.Fatal(e)
				}
				code = "E_RECOVERY_REQUIRED"
			case "stale-manifest":
				s.Manifest = append(s.Manifest, ' ')
			case "stale-payload":
				v := s.Entries[ownerNameLockFile("peer")]
				v.Data = append(append([]byte(nil), v.Data...), ' ')
				s.Entries[ownerNameLockFile("peer")] = v
			}
			a, b := snapshot(t, src), snapshot(t, dst)
			for _, check := range []bool{true, false} {
				_, e := MoveOutputRoot(src, dst, s, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
					t.Fatal("refusal changed output")
				}
			}
		})
	}
}

func TestRootMoveEveryPrepareApplyBoundaryRollback(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	a, b := snapshot(t, src), snapshot(t, dst)
	// Each root journals every owned entry exactly once, with manifest last.
	for ri := 0; ri < 2; ri++ {
		for op := 0; op < len(s.Entries); op++ {
			_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, operation int, stage string) error {
				if phase == "apply" {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if j.Operations[len(j.Operations)-1].Path != manifestName {
						t.Fatal("manifest not last")
					}
				}
				if index == ri && phase == "apply" && operation == op {
					return errors.New("synthetic fault")
				}
				return nil
			})
			errorCode(t, e, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("rollback lost bytes/mode/mtime", ri, op)
			}
		}
		_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, _ int, _ string) error {
			if index == ri && phase == "prepare" {
				return errors.New("synthetic preparation failure")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_IO")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("prepare failure applied root")
		}
	}
}

func TestRootMoveCASUnderCompleteLocksAndAbsentReadonlyTarget(t *testing.T) {
	for _, kind := range []string{"manifest", "file-mode", "target", "source-recovery", "target-recovery"} {
		src, dst, s := moveFixture(t, 3)
		var a, b map[string]snapshotEntry
		_, e := moveOutputRoot(src, dst, s, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" {
				switch kind {
				case "manifest":
					put(t, filepath.Join(src, manifestName), append(s.Manifest, ' '))
				case "file-mode":
					if e := os.Chmod(filepath.Join(src, s.Files[0].Path), 0444); e != nil {
						t.Fatal(e)
					}
				case "target":
					put(t, filepath.Join(dst, manifestName), s.Manifest)
				default:
					root := src
					if kind == "target-recovery" {
						root = dst
					}
					if e := os.Mkdir(filepath.Join(root, stagePrefix+"interrupted"), 0700); e != nil {
						t.Fatal(e)
					}
				}
				a, b = snapshot(t, src), snapshot(t, dst)
			}
			return nil
		})
		code := "E_OUTPUT_ROOT_MOVE_STALE"
		if kind == "target" {
			code = "E_OUTPUT_ROOT_MOVE_OCCUPIED"
		}
		if kind == "source-recovery" || kind == "target-recovery" {
			code = "E_RECOVERY_REQUIRED"
		}
		errorCode(t, e, code)
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("CAS lost intervening state")
		}
	}
	src, _, s := moveFixture(t, 1)
	dst := filepath.Join(t.TempDir(), "absent", "target")
	if changed, e := MoveOutputRoot(src, dst, s, true); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if _, e := os.Stat(filepath.Dir(dst)); !os.IsNotExist(e) {
		t.Fatal("readonly created parents")
	}
}
