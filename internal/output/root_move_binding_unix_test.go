//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveAncestorBindingCapturedBeforeGenerationAndLockedCAS(t *testing.T) {
	src, dst, _ := moveFixture(t, 1)
	parent := t.TempDir()
	alias := filepath.Join(parent, "alias")
	if e := os.Symlink(filepath.Dir(src), alias); e != nil {
		t.Fatal(e)
	}
	raw := filepath.Join(alias, filepath.Base(src))
	s, e := ReadRootMoveState(raw)
	if e != nil {
		t.Fatal(e)
	}
	alternate := filepath.Join(parent, "alternate")
	if e := os.MkdirAll(filepath.Join(alternate, filepath.Base(src)), 0700); e != nil {
		t.Fatal(e)
	}
	for path, entry := range s.Entries {
		dest := filepath.Join(alternate, filepath.Base(src), path)
		if e := os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(dest, entry.Data, entry.Attributes.Mode); e != nil {
			t.Fatal(e)
		}
		if e := os.Chtimes(dest, entry.Attributes.Modified, entry.Attributes.Modified); e != nil {
			t.Fatal(e)
		}
	}
	for _, when := range []string{"before-call", "prelock"} {
		if e := os.Remove(alias); e != nil {
			t.Fatal(e)
		}
		if e := os.Symlink(filepath.Dir(src), alias); e != nil {
			t.Fatal(e)
		}
		rebind := func() {
			if e := os.Remove(alias); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(alternate, alias); e != nil {
				t.Fatal(e)
			}
		}
		if when == "before-call" {
			rebind()
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		_, e := moveOutputRoot(raw, dst, s, false, func(_ int, phase string, _ int, _ string) error {
			if phase == "prelock" && when == "prelock" {
				rebind()
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_ROOT_MOVE_STALE")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("binding CAS mutated roots")
		}
	}
}

func TestRootMoveMetadataBackupAndTargetRollbackEvidence(t *testing.T) {
	for _, which := range []string{"source-lock", "source-manifest", "target-lock", "target-manifest"} {
		t.Run(which, func(t *testing.T) {
			src, dst, s := moveFixture(t, 3)
			targetFailure := strings.HasPrefix(which, "target")
			if targetFailure {
				nextSrc, nextDst := filepath.Join(filepath.Dir(src), "z-source"), filepath.Join(filepath.Dir(src), "a-target")
				if e := os.Rename(src, nextSrc); e != nil {
					t.Fatal(e)
				}
				if e := os.Rename(dst, nextDst); e != nil {
					t.Fatal(e)
				}
				src, dst = nextSrc, nextDst
				current, e := ReadRootMoveState(src)
				if e != nil {
					t.Fatal(e)
				}
				s = current
			}
			path := nameLockFile
			if strings.HasSuffix(which, "manifest") {
				path = manifestName
			}
			obstructed := src
			other := dst
			if targetFailure {
				obstructed = dst
				other = src
			}
			oldOther := snapshot(t, other)
			_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, op int, _ string) error {
				if phase == "apply" && index == 1 && op == len(s.Entries)-1 {
					obstructLifecycleRollback(t, obstructed, path)
					return errors.New("metadata restoration obstruction")
				}
				return nil
			})
			errorCode(t, e, "E_RECOVERY_REQUIRED")
			if !reflect.DeepEqual(oldOther, snapshot(t, other)) {
				t.Fatal("other root rollback skipped")
			}
			if !targetFailure {
				requireLifecycleBackup(t, src, path, s.Entries[path].Data)
			} else {
				entries, err := os.ReadDir(dst)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), stagePrefix) {
						var j journal
						if err := json.Unmarshal(mustRead(t, filepath.Join(dst, entry.Name(), "transaction.json")), &j); err != nil {
							t.Fatal(err)
						}
						for _, op := range j.Operations {
							if op.Path == path && !op.HadOld {
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatal("target obstruction journal absent")
				}
			}
		})
	}
}
