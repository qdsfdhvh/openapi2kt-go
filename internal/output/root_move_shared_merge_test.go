package output

import (
	"encoding/json"
	"errors"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRootMoveSharedMergeProfilesKeepOriginalTargetAttributes(t *testing.T) {
	for _, version := range []int{2, 4} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, owner := range []string{"", "alpha", "peer"} {
				src, dst, s, target := mergeMoveFixtureForPaths(t, version, style, true)
				a, b := snapshot(t, src), snapshot(t, dst)
				shared := map[string]bool{}
				for _, f := range s.Files {
					for _, g := range target.Files {
						if f.Path == g.Path {
							if f.Content != g.Content {
								t.Fatal("bad shared fixture")
							}
							if reflect.DeepEqual(a[filepath.FromSlash(f.Path)], b[filepath.FromSlash(f.Path)]) {
								t.Fatal("fixture must vary source and target attributes")
							}
							shared[f.Path] = true
						}
					}
				}
				if len(shared) == 0 {
					t.Fatal("shared fixture has no same-path bytes")
				}
				var owners []string
				if owner != "" {
					owners = []string{owner}
				}
				var e *core.Diagnostic
				if owner != "" {
					s, e = BindRootMoveOwnerTarget(s, dst, src, owners)
				} else {
					s, e = BindRootMoveTarget(s, dst, src)
				}
				if e != nil {
					t.Fatal("identical owned Kotlin should merge while retaining target attributes", version, style, owner, e)
				}
				for _, check := range []bool{true, false} {
					changed, e := moveOutputSelection(src, dst, s, owners, check, nil)
					if e != nil || !changed {
						t.Fatal(changed, e)
					}
					if check {
						if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
							t.Fatal("shared readonly wrote")
						}
						continue
					}
					sourceAfter, targetAfter := snapshot(t, src), snapshot(t, dst)
					for path := range target.Entries {
						if path != manifestName && !reflect.DeepEqual(b[filepath.FromSlash(path)], targetAfter[filepath.FromSlash(path)]) {
							t.Fatal("shared merge rewrote original target", path)
						}
					}
					for name, paths := range s.Owners {
						if owner != "" && name != owner {
							for _, path := range paths {
								if !reflect.DeepEqual(a[filepath.FromSlash(path)], sourceAfter[filepath.FromSlash(path)]) {
									t.Fatal("source peer changed", path)
								}
							}
							continue
						}
						for _, path := range paths {
							want := a[filepath.FromSlash(path)]
							if shared[path] {
								want = b[filepath.FromSlash(path)]
							}
							if !reflect.DeepEqual(want, targetAfter[filepath.FromSlash(path)]) {
								t.Fatal("wrong shared/new target attribute authority", path)
							}
						}
						if _, locked := s.NameLocks[name]; locked && !reflect.DeepEqual(a[filepath.FromSlash(ownerNameLockFile(name))], targetAfter[filepath.FromSlash(ownerNameLockFile(name))]) {
							t.Fatal("moved raw lock changed")
						}
					}
					result, e := ReadRootMoveState(dst)
					if e != nil {
						t.Fatal(e)
					}
					wanted := len(target.Owners) + len(s.Owners)
					if owner != "" {
						wanted = len(target.Owners) + 1
					}
					if len(result.Owners) != wanted {
						t.Fatal("shared owner union lost membership")
					}
				}
			}
		}
	}
}

func TestRootMoveSharedMergeEveryBoundaryRollback(t *testing.T) {
	for _, selected := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			src, dst, s, _ := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
			if reverse {
				a, b := filepath.Join(filepath.Dir(src), "z-source"), filepath.Join(filepath.Dir(dst), "a-target")
				if e := os.Rename(src, a); e != nil {
					t.Fatal(e)
				}
				if e := os.Rename(dst, b); e != nil {
					t.Fatal(e)
				}
				src, dst = a, b
				s, _ = ReadRootMoveState(src)
			}
			var owners []string
			if selected {
				owners = []string{"alpha"}
			}
			var e *core.Diagnostic
			if selected {
				s, e = BindRootMoveOwnerTarget(s, dst, src, owners)
			} else {
				s, e = BindRootMoveTarget(s, dst, src)
			}
			if e != nil {
				t.Fatal(e)
			}
			a, b := snapshot(t, src), snapshot(t, dst)
			// Discover the actual journals, requiring manifests last and every root
			// prepared before the first apply. Inject at every discovered operation.
			counts := map[int]int{}
			prepared := []int{}
			_, e = moveOutputSelection(src, dst, s, owners, false, func(ri int, phase string, op int, stage string) error {
				if phase == "prepare" {
					prepared = append(prepared, ri)
				}
				if phase == "apply" {
					if !reflect.DeepEqual(prepared, []int{0, 1}) {
						t.Fatal("apply before complete staging")
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
						return errors.New("discover final failure")
					}
				}
				return nil
			})
			errorCode(t, e, "E_OUTPUT_COMMIT")
			for ri := 0; ri < 2; ri++ {
				for op := 0; op < counts[ri]; op++ {
					_, e := moveOutputSelection(src, dst, s, owners, false, func(index int, phase string, at int, _ string) error {
						if index == ri && phase == "apply" && at == op {
							return errors.New("synthetic merge failure")
						}
						return nil
					})
					errorCode(t, e, "E_OUTPUT_COMMIT")
					if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
						t.Fatal("merge rollback lost state", selected, reverse, ri, op)
					}
				}
				_, e := moveOutputSelection(src, dst, s, owners, false, func(index int, phase string, _ int, _ string) error {
					if index == ri && phase == "prepare" {
						return errors.New("synthetic merge staging failure")
					}
					return nil
				})
				errorCode(t, e, "E_OUTPUT_IO")
				if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
					t.Fatal("merge staging changed state")
				}
			}
		}
	}
}

func TestRootMoveSharedMergeBothSideCAS(t *testing.T) {
	for _, side := range []string{"source", "target"} {
		for _, kind := range []string{"manifest", "code", "lock", "mode", "mtime", "valid-lock-change", "foreign", "recovery"} {
			src, dst, s, target := mergeMoveFixtureForPaths(t, 4, core.DataClass, true)
			s, e := BindRootMoveOwnerTarget(s, dst, src, []string{"alpha"})
			if e != nil {
				t.Fatal(e)
			}
			root, state, owner := dst, target, "beta"
			if side == "source" {
				root, state, owner = src, s, "alpha"
			}
			a, b := snapshot(t, src), snapshot(t, dst)
			code := "E_OUTPUT_ROOT_MOVE_STALE"
			_, e = moveOutputSelection(src, dst, s, []string{"alpha"}, false, func(_ int, phase string, _ int, _ string) error {
				if phase != "prelock" {
					return nil
				}
				path := filepath.Join(root, state.Files[0].Path)
				switch kind {
				case "manifest":
					put(t, filepath.Join(root, manifestName), append(state.Manifest, ' '))
				case "code":
					put(t, path, []byte("// intervening payload"))
					code = "E_OUTPUT_MODIFIED"
				case "lock":
					put(t, filepath.Join(root, ownerNameLockFile(owner)), []byte("edited"))
					code = "E_NAME_LOCK_MODIFIED"
				case "mode":
					if e := os.Chmod(path, 0444); e != nil {
						t.Fatal(e)
					}
				case "mtime":
					stamp := time.Unix(1400000000, 0)
					if e := os.Chtimes(path, stamp, stamp); e != nil {
						t.Fatal(e)
					}
				case "valid-lock-change":
					data := append(state.NameLocks[owner], ' ')
					put(t, filepath.Join(root, ownerNameLockFile(owner)), data)
					m, e := readManifestKind(root, true, true)
					if e != nil {
						t.Fatal(e)
					}
					(*m.NameLocks)[owner] = digest(data)
					raw, _ := json.MarshalIndent(m, "", "  ")
					put(t, filepath.Join(root, manifestName), raw)
				case "foreign":
					put(t, filepath.Join(root, nameLockFile), []byte("foreign"))
					code = "E_OUTPUT_ROOT_MOVE_INVALID"
				case "recovery":
					if e := os.Mkdir(filepath.Join(root, stagePrefix+"foreign"), 0700); e != nil {
						t.Fatal(e)
					}
					code = "E_RECOVERY_REQUIRED"
				}
				a, b = snapshot(t, src), snapshot(t, dst)
				return nil
			})
			errorCode(t, e, code)
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("merge CAS destroyed intervening state", side, kind)
			}
		}
	}
}
