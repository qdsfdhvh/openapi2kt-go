package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"openapi-kt/core"
	"openapi-kt/internal/schema"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func mergeMoveFixture(t testing.TB, version int, style core.ModelStyle) (string, string, RootMoveState, RootMoveState) {
	return mergeMoveFixtureForPaths(t, version, style, false)
}

func mergeMoveFixtureForPaths(t testing.TB, version int, style core.ModelStyle, sharedTarget bool) (string, string, RootMoveState, RootMoveState) {
	t.Helper()
	base := t.TempDir()
	src, dst := filepath.Join(base, "a-source"), filepath.Join(base, "z-target")
	for i, root := range []string{src, dst} {
		names := []string{"alpha", "peer"}
		if i == 1 {
			names = []string{"beta", "gamma"}
		}
		generated := map[string]core.GenerationArtifacts{}
		previous := map[string][]byte{}
		for n, name := range names {
			locked := version == 4 && n == 0
			request := ownerLockRequest(t, name, style, locked)
			if sharedTarget {
				request.Options.PackageName = "org.example.sharedroot"
			}
			generated[name] = singleGenerated(t, request)
			if i == 0 {
				g := generated[name]
				g.Files = append(g.Files, core.GeneratedFile{Path: "source/Shared.kt", Content: "package source\nobject Shared\n"})
				generated[name] = g
			}
			if locked {
				previous[name] = nil
			}
		}
		if _, e := WriteNameLockedOwners(root, lockedUpdate(names, generated, previous, true), false); e != nil {
			t.Fatal(e)
		}
		// Preserve accepted noncanonical payload serialization independently of the
		// canonical manifest that must change to record merged membership.
		if version == 4 {
			path := ownerNameLockFile(names[0])
			data := append(mustRead(t, filepath.Join(root, path)), '\n')
			put(t, filepath.Join(root, path), data)
			m, e := readManifestKind(root, true, true)
			if e != nil {
				t.Fatal(e)
			}
			(*m.NameLocks)[names[0]] = digest(data)
			raw, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(root, manifestName), append(raw, '\n'))
		}
		put(t, filepath.Join(root, "Notes.kt"), []byte("// handwritten "+names[0]))
		for path, entry := range snapshot(t, root) {
			if entry.Directory || path == lockName {
				continue
			}
			full := filepath.Join(root, path)
			if e := os.Chmod(full, 0640); e != nil {
				t.Fatal(e)
			}
			stamp := time.Unix(1200000000+int64(i), 0)
			if e := os.Chtimes(full, stamp, stamp); e != nil {
				t.Fatal(e)
			}
		}
	}
	a, e := ReadRootMoveState(src)
	if e != nil {
		t.Fatal(e)
	}
	b, e := ReadRootMoveState(dst)
	if e != nil {
		t.Fatal(e)
	}
	return src, dst, a, b
}

func TestRootMoveMergeFullAndSelectedPreserveInventories(t *testing.T) {
	for _, version := range []int{2, 4} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, selected := range []bool{false, true} {
				src, dst, s, target := mergeMoveFixture(t, version, style)
				a, b := snapshot(t, src), snapshot(t, dst)
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
					t.Fatal("disjoint owned target was rejected", version, style, selected, e)
				}
				for _, check := range []bool{true, false} {
					changed, e := moveOutputSelection(src, dst, s, owners, check, nil)
					if e != nil || !changed {
						t.Fatal(changed, e)
					}
					if check {
						if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
							t.Fatal("readonly merge wrote")
						}
						continue
					}
					result, e := ReadRootMoveState(dst)
					if e != nil {
						t.Fatal(e)
					}
					wanted := len(s.Owners) + len(target.Owners)
					if selected {
						wanted = 1 + len(target.Owners)
					}
					if result.Version != s.Version || len(result.Owners) != wanted {
						t.Fatal("lost merged membership")
					}
					for path := range target.Entries {
						if path != manifestName && !reflect.DeepEqual(b[path], snapshot(t, dst)[path]) {
							t.Fatal("changed target payload", path)
						}
					}
					for owner, paths := range s.Owners {
						moved := !selected || owner == "alpha"
						root := src
						if moved {
							root = dst
						}
						for _, path := range paths {
							if !reflect.DeepEqual(a[path], snapshot(t, root)[path]) {
								t.Fatal("changed source payload", owner, path)
							}
							if moved && !(selected && path == "source/Shared.kt") {
								if _, exists := snapshot(t, src)[path]; exists {
									t.Fatal("exclusive moved source survives", path)
								}
							}
						}
						if lock, ok := s.NameLocks[owner]; ok {
							if !reflect.DeepEqual(lock, mustRead(t, filepath.Join(root, ownerNameLockFile(owner)))) {
								t.Fatal("raw lock rewritten")
							}
						}
					}
					if !reflect.DeepEqual(a["Notes.kt"], snapshot(t, src)["Notes.kt"]) || !reflect.DeepEqual(b["Notes.kt"], snapshot(t, dst)["Notes.kt"]) {
						t.Fatal("unowned file changed")
					}
				}
			}
		}
	}
}

func TestRootMoveMergeEveryBoundaryRollback(t *testing.T) {
	for _, selected := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			src, dst, s, _ := mergeMoveFixture(t, 4, core.DataClass)
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

func TestRootMoveMergeBothSideCAS(t *testing.T) {
	for _, side := range []string{"source", "target"} {
		for _, kind := range []string{"manifest", "code", "lock", "mode", "mtime", "valid-lock-change", "foreign", "recovery"} {
			src, dst, s, target := mergeMoveFixture(t, 4, core.DataClass)
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
					if e := os.Chmod(path, 0600); e != nil {
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

func TestRootMoveMergeCollisionAndVersionRefusals(t *testing.T) {
	for _, kind := range []string{"owner", "version", "exact-path", "changed-path", "case-path", "unowned", "parent", "foreign", "modified", "missing"} {
		src, dst, s, target := mergeMoveFixture(t, 4, core.DataClass)
		code := "E_OUTPUT_ROOT_MOVE_OCCUPIED"
		switch kind {
		case "owner":
			m, e := readManifestKind(dst, true, true)
			if e != nil {
				t.Fatal(e)
			}
			(*m.Owners)["alpha"] = (*m.Owners)["gamma"]
			delete(*m.Owners, "gamma")
			raw, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(dst, manifestName), raw)
		case "version":
			// Explicitly construct the valid v2 representation after removing only its
			// known owned metadata; the action itself must never do version adoption.
			m, e := readManifestKind(dst, true, true)
			if e != nil {
				t.Fatal(e)
			}
			m.Version = 2
			m.NameLocks = nil
			raw, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(dst, manifestName), raw)
			if e := os.Remove(filepath.Join(dst, ownerNameLockFile("beta"))); e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
		case "exact-path", "changed-path", "case-path":
			f := s.Files[0]
			if kind == "changed-path" {
				f.Content += "\n// different original target bytes\n"
			}
			path := f.Path
			if kind == "case-path" {
				path = strings.ToUpper(strings.TrimSuffix(path, ".kt")) + ".kt"
			}
			m, e := readManifestKind(dst, true, true)
			if e != nil {
				t.Fatal(e)
			}
			m.Files[path] = digest([]byte(f.Content))
			(*m.Owners)["gamma"] = append((*m.Owners)["gamma"], path)
			sort.Strings((*m.Owners)["gamma"])
			raw, _ := json.MarshalIndent(m, "", "  ")
			put(t, filepath.Join(dst, manifestName), raw)
			if e := os.MkdirAll(filepath.Dir(filepath.Join(dst, path)), 0700); e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(dst, path), []byte(f.Content))
		case "unowned":
			path := s.Files[0].Path
			if e := os.MkdirAll(filepath.Dir(filepath.Join(dst, path)), 0700); e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(dst, path), []byte(s.Files[0].Content))
			code = "E_OUTPUT_COLLISION"
		case "parent":
			path := filepath.Dir(s.Files[0].Path)
			if e := os.MkdirAll(filepath.Dir(filepath.Join(dst, path)), 0700); e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(dst, path), []byte("handwritten parent"))
			code = "E_OUTPUT_PATH"
		case "foreign":
			put(t, filepath.Join(dst, nameLockFile), []byte("foreign"))
			code = "E_OUTPUT_ROOT_MOVE_INVALID"
		case "modified":
			put(t, filepath.Join(dst, target.Files[0].Path), []byte("edited"))
			code = "E_OUTPUT_MODIFIED"
		case "missing":
			if e := os.Remove(filepath.Join(dst, target.Files[0].Path)); e != nil {
				t.Fatal(e)
			}
			code = "E_OUTPUT_MISSING"
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		bound, e := BindRootMoveTarget(s, dst, src)
		if kind == "exact-path" {
			// Keep the exact original96 single-path refusal input as an explicit
			// shared-membership positive; its target payload remains untouched.
			if e != nil {
				t.Fatal("original exact-path input still refused", e)
			}
			if changed, e := MoveOutputRoot(src, dst, bound, true); e != nil || !changed {
				t.Fatal(changed, e)
			}
		} else {
			errorCode(t, e, code)
		}
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("merge refusal wrote", kind)
		}
	}
}

func TestRootMoveMergeCombinedLockBudgets(t *testing.T) {
	for _, kind := range []string{"count", "bytes"} {
		for _, over := range []bool{false, true} {
			base := t.TempDir()
			src, dst := filepath.Join(base, "source"), filepath.Join(base, "target")
			for side, root := range []string{src, dst} {
				count := 128
				if kind == "bytes" {
					count = 8
				}
				if over && side == 0 {
					count++
				}
				artifact := singleGenerated(t, ownerLockRequest(t, fmt.Sprintf("side%d", side), core.DataClass, true))
				names := []string{}
				generated := map[string]core.GenerationArtifacts{}
				previous := map[string][]byte{}
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("owner%d_%d", side, i)
					names = append(names, name)
					generated[name] = artifact
					previous[name] = nil
				}
				if _, e := WriteNameLockedOwners(root, lockedUpdate(names, generated, previous, true), false); e != nil {
					t.Fatal(e)
				}
				if kind == "bytes" {
					data := append(append([]byte{}, artifact.NameLock...), bytes.Repeat([]byte(" "), schema.MaxNameLockBytes-len(artifact.NameLock))...)
					m, e := readManifestKind(root, true, true)
					if e != nil {
						t.Fatal(e)
					}
					for _, name := range names {
						put(t, filepath.Join(root, ownerNameLockFile(name)), data)
						(*m.NameLocks)[name] = digest(data)
					}
					raw, _ := json.MarshalIndent(m, "", "  ")
					put(t, filepath.Join(root, manifestName), raw)
				}
			}
			s, e := ReadRootMoveState(src)
			if e != nil {
				t.Fatal(e)
			}
			a, b := snapshot(t, src), snapshot(t, dst)
			s, e = BindRootMoveTarget(s, dst, src)
			if over {
				errorCode(t, e, "E_NAME_LOCK_LIMIT")
			} else {
				if e != nil {
					t.Fatal(kind, e)
				}
				if changed, e := MoveOutputRoot(src, dst, s, true); e != nil || !changed {
					t.Fatal("exact combined budget failed", kind, changed, e)
				}
			}
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("combined budget preflight wrote", kind, over)
			}
		}
	}
}

func TestRootMoveMergeSelectedUnlockedOwnerRetainsBothLockInventories(t *testing.T) {
	src, dst, s, target := mergeMoveFixture(t, 4, core.PlainClass)
	s, e := BindRootMoveOwnerTarget(s, dst, src, []string{"peer"})
	if e != nil {
		t.Fatal(e)
	}
	if changed, e := MoveOutputOwners(src, dst, s, []string{"peer"}, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
	source, e := ReadRootMoveState(src)
	if e != nil {
		t.Fatal(e)
	}
	result, e := ReadRootMoveState(dst)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(source.NameLocks, s.NameLocks) || !reflect.DeepEqual(result.NameLocks, target.NameLocks) || result.Owners["peer"] == nil || source.Owners["alpha"] == nil {
		t.Fatal("unlocked move changed original lock modes or peer membership")
	}
}
