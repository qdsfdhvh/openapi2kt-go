package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type multipleMoveManifest struct {
	Version   int                 `json:"version"`
	Files     map[string]string   `json:"files"`
	Owners    map[string][]string `json:"owners"`
	NameLocks map[string]string   `json:"name_locks,omitempty"`
}

func multipleMoveState(t testing.TB, root string) multipleMoveManifest {
	t.Helper()
	var m multipleMoveManifest
	if e := json.Unmarshal(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), &m); e != nil {
		t.Fatal(e)
	}
	return m
}

// Three source owners have exclusive models as well as identical shared Kotlin.
// A target owner and a disjoint config peer exercise storage-only/excluded IO.
func multipleMoveFixture(t testing.TB, mode string, owned bool) (string, string, string, string, string) {
	t.Helper()
	base := t.TempDir()
	names := []string{"a", "b", "c", "d", "x"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, false, false, false)
	text := string(read(t, cfg))
	for _, name := range names {
		data := read(t, filepath.Join(base, name+".json"))
		var doc map[string]any
		if e := json.Unmarshal(data, &doc); e != nil {
			t.Fatal(e)
		}
		if doc["components"] == nil {
			doc["components"] = map[string]any{}
		}
		components := doc["components"].(map[string]any)
		if components["schemas"] == nil {
			components["schemas"] = map[string]any{}
		}
		components["schemas"].(map[string]any)["Only"+strings.ToUpper(name)] = map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}}
		data, e := json.Marshal(doc)
		if e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(base, name+".json"), data)
		locked := mode == "locked" || mode == "mixed" && (name == "a" || name == "c" || name == "d") || mode == "retained-lock" && (name == "c" || name == "d")
		text = strings.Replace(text, "name="+strconv.Quote(name), "name="+strconv.Quote(name)+"\nlock_hoisted_names="+strconv.FormatBool(locked), 1)
		text = strings.Replace(text, "org.example.lock"+name+"\"", "org.example.shared\"", 1)
	}
	if owned {
		text = strings.Replace(text, "name=\"d\"", "name=\"d\"\noutput_dir='target'", 1)
	} else {
		text = strings.Replace(text, "name=\"d\"", "name=\"d\"\noutput_dir='elsewhere'", 1)
	}
	text = strings.Replace(text, "name=\"x\"", "name=\"x\"\noutput_dir='unrelated'", 1)
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	src, dst := filepath.Join(base, "generated"), filepath.Join(base, "target")
	roots := []string{src}
	if owned {
		roots = append(roots, dst)
	}
	for _, root := range roots {
		m := multipleMoveState(t, root)
		// Manifest formatting is intentionally noncanonical; canonical locks permit
		// the later ordinary-generation no-op control. A separate test keeps raw locks.
		raw, e := json.MarshalIndent(m, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(root, ".openapi-kt-manifest.json"), append(raw, ' ', '\n'))
		for path := range snapshot(t, root) {
			if path == ".openapi-kt.lock" {
				continue
			}
			full := filepath.Join(root, path)
			if e := os.Chmod(full, 0640); e != nil {
				t.Fatal(e)
			}
			stamp := time.Unix(1200000000, 0)
			if root == dst {
				stamp = stamp.Add(time.Hour)
			}
			if e := os.Chtimes(full, stamp, stamp); e != nil {
				t.Fatal(e)
			}
		}
		put(t, filepath.Join(root, "Notes.kt"), []byte("// handwritten\n"))
	}
	return base, cfg, src, dst, text
}

func TestRootMoveMultipleOwnersNativeSubset(t *testing.T) {
	for _, mode := range []string{"unlocked", "locked", "mixed", "retained-lock"} {
		for _, owned := range []bool{false, true} {
			t.Run(mode+"/owned="+strconv.FormatBool(owned), func(t *testing.T) {
				base, cfg, src, dst, text := multipleMoveFixture(t, mode, owned)
				original := multipleMoveState(t, src)
				before := snapshot(t, src)
				peer := snapshot(t, filepath.Join(base, "unrelated"))
				var targetBefore map[string]snap
				if owned {
					targetBefore = snapshot(t, dst)
				}
				// Missing excluded inputs must not be read, including original target peers.
				d, x := read(t, filepath.Join(base, "d.json")), read(t, filepath.Join(base, "x.json"))
				for _, name := range []string{"d", "x"} {
					if e := os.Remove(filepath.Join(base, name+".json")); e != nil {
						t.Fatal(e)
					}
				}
				args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst, "--move-owner=b", "--move-owner=a"}
				checked := invoke(t, base, append(append([]string{}, args...), "--check")...)
				result(t, checked, 3)
				validated := invoke(t, base, lifecycleValidate(args)...)
				result(t, validated, 0)
				// Order has no semantic/report effect.
				reversed := append([]string{}, args[:len(args)-2]...)
				reversed = append(reversed, "--move-owner=a", "--move-owner=b", "--check")
				reordered := invoke(t, base, reversed...)
				result(t, reordered, 3)
				if !bytes.Equal(checked.Stdout, reordered.Stdout) {
					t.Fatal("selection order changed report")
				}
				if !reflect.DeepEqual(before, snapshot(t, src)) || !reflect.DeepEqual(peer, snapshot(t, filepath.Join(base, "unrelated"))) {
					t.Fatal("readonly move wrote source/peer")
				}
				if owned {
					if !reflect.DeepEqual(targetBefore, snapshot(t, dst)) {
						t.Fatal("readonly target changed")
					}
				} else if _, e := os.Lstat(dst); !os.IsNotExist(e) {
					t.Fatal("readonly created target", e)
				}
				applied := invoke(t, base, args...)
				result(t, applied, 0)
				wanted := map[string]bool{}
				for _, name := range []string{"a", "b"} {
					for _, p := range original.Owners[name] {
						wanted[p] = true
					}
				}
				paths := []string{}
				for p := range wanted {
					paths = append(paths, p)
				}
				sort.Strings(paths)
				for _, out := range []invocation{checked, validated, applied} {
					var raw struct {
						Files     []string
						Specs     []struct{ Name string }
						Migration struct {
							Moved    []string `json:"moved_owners"`
							Retained []string `json:"retained_owners"`
							Files    []struct {
								Before, After, SHA256 string
								Owners                []string
							}
							Locks  []struct{ Owner, Path, SHA256 string } `json:"name_locks"`
							Hash   string                                 `json:"manifest_sha256"`
							Target []string                               `json:"target_owners"`
						} `json:"output_root_migration"`
					}
					if e := json.Unmarshal(out.Stdout, &raw); e != nil {
						t.Fatal(e)
					}
					sum := sha256.Sum256([]byte(before[".openapi-kt-manifest.json"].Bytes))
					if !reflect.DeepEqual(raw.Migration.Moved, []string{"a", "b"}) || !reflect.DeepEqual(raw.Migration.Retained, []string{"c"}) || !reflect.DeepEqual(raw.Files, paths) || len(raw.Migration.Files) != len(paths) || raw.Migration.Hash != hex.EncodeToString(sum[:]) || len(raw.Specs) != 3 || bytes.Contains(out.Stdout, []byte(base)) {
						t.Fatal("selection/full-source report", string(out.Stdout))
					}
					if owned && !reflect.DeepEqual(raw.Migration.Target, []string{"d"}) {
						t.Fatal("target inventory")
					}
					for _, f := range raw.Migration.Files {
						authority := []string{}
						for _, owner := range []string{"a", "b", "c"} {
							for _, p := range original.Owners[owner] {
								if p == f.Before {
									authority = append(authority, owner)
								}
							}
						}
						if f.Before != f.After || !reflect.DeepEqual(f.Owners, authority) || original.Files[f.Before] != f.SHA256 {
							t.Fatal("lost original shared owner authority", f)
						}
					}
					count := 0
					for _, owner := range []string{"a", "b"} {
						if _, ok := original.NameLocks[owner]; ok {
							count++
						}
					}
					if len(raw.Migration.Locks) != count {
						t.Fatal("lock union")
					}
					for _, lock := range raw.Migration.Locks {
						if lock.SHA256 != original.NameLocks[lock.Owner] || lock.Path != namedLockPath(lock.Owner) {
							t.Fatal("lock report")
						}
					}
				}
				afterSrc, afterDst := snapshot(t, src), snapshot(t, dst)
				source, target := multipleMoveState(t, src), multipleMoveState(t, dst)
				if source.Version != original.Version || target.Version != original.Version || len(source.Owners) != 1 || !reflect.DeepEqual(source.Owners["c"], original.Owners["c"]) || !reflect.DeepEqual(target.Owners["a"], original.Owners["a"]) || !reflect.DeepEqual(target.Owners["b"], original.Owners["b"]) {
					t.Fatal("manifest union")
				}
				expectedFiles, expectedLocks := map[string]string{}, map[string]string{}
				for path := range wanted {
					expectedFiles[path] = original.Files[path]
				}
				for _, owner := range []string{"a", "b"} {
					if hash, ok := original.NameLocks[owner]; ok {
						expectedLocks[owner] = hash
					}
				}
				expectedOwners := 2
				if owned {
					expectedOwners++
					var oldTarget multipleMoveManifest
					if e := json.Unmarshal([]byte(targetBefore[".openapi-kt-manifest.json"].Bytes), &oldTarget); e != nil {
						t.Fatal(e)
					}
					for path, hash := range oldTarget.Files {
						expectedFiles[path] = hash
					}
					for owner, hash := range oldTarget.NameLocks {
						expectedLocks[owner] = hash
					}
				}
				if len(target.Owners) != expectedOwners || !reflect.DeepEqual(target.Files, expectedFiles) || len(target.NameLocks) != len(expectedLocks) {
					t.Fatal("target owner/file/lock union differs")
				}
				for owner, hash := range expectedLocks {
					if target.NameLocks[owner] != hash {
						t.Fatal("target lock hash union differs", owner)
					}
				}
				if len(source.Files) != len(original.Owners["c"]) {
					t.Fatal("retained source file inventory differs")
				}
				retainedLocks := 0
				if hash, locked := original.NameLocks["c"]; locked {
					retainedLocks = 1
					if source.NameLocks["c"] != hash {
						t.Fatal("retained source lock hash changed")
					}
				}
				if len(source.NameLocks) != retainedLocks {
					t.Fatal("retained source lock inventory differs")
				}
				for _, owner := range []string{"a", "b", "c"} {
					actual := afterDst
					if owner == "c" {
						actual = afterSrc
					}
					for _, p := range original.Owners[owner] {
						want := before[p]
						if owner != "c" {
							if old, shared := targetBefore[p]; shared {
								want = old
							}
						}
						if !reflect.DeepEqual(want, actual[p]) {
							t.Fatal("payload/attribute changed", owner, p)
						}
						if owner != "c" && !containsMovePath(original.Owners["c"], p) {
							if _, exists := afterSrc[p]; exists {
								t.Fatal("exclusive source payload retained", p)
							}
						}
					}
					if _, ok := original.NameLocks[owner]; ok {
						if !reflect.DeepEqual(before[namedLockPath(owner)], actual[namedLockPath(owner)]) {
							t.Fatal("raw lock changed", owner)
						}
					}
				}
				for p, old := range targetBefore {
					if p != ".openapi-kt-manifest.json" && !reflect.DeepEqual(old, afterDst[p]) {
						t.Fatal("target peer changed", p)
					}
				}
				if !reflect.DeepEqual(before["Notes.kt"], afterSrc["Notes.kt"]) || !reflect.DeepEqual(peer, snapshot(t, filepath.Join(base, "unrelated"))) {
					t.Fatal("unowned/excluded peer changed")
				}
				for _, owner := range []string{"a", "b"} {
					text = strings.Replace(text, "name="+strconv.Quote(owner), "name="+strconv.Quote(owner)+"\noutput_dir='target'", 1)
				}
				put(t, cfg, []byte(text))
				put(t, filepath.Join(base, "d.json"), d)
				put(t, filepath.Join(base, "x.json"), x)
				result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json", "--check"), 0)
				if !reflect.DeepEqual(afterSrc, snapshot(t, src)) || !reflect.DeepEqual(afterDst, snapshot(t, dst)) {
					t.Fatal("ordinary check wrote")
				}
			})
		}
	}
}

func containsMovePath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func TestRootMoveMultipleOwnersNativeGuards(t *testing.T) {
	base, cfg, src, dst, text := multipleMoveFixture(t, "mixed", false)
	before := snapshot(t, src)
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst}
	for _, extra := range [][]string{{"--move-owner="}, {"--move-owner=1bad"}, {"--move-owner=a,b"}, {"--move-owner=å"}, {"--move-owner=a", "--move-owner=a"}, {"--move-owner=a", "--spec=a"}, {"--spec=a", "--spec=b"}} {
		fails(t, invoke(t, base, append(append([]string{}, args...), extra...)...), 2, "E_USAGE")
	}
	for _, owner := range []string{"unknown", "x", "d"} {
		fails(t, invoke(t, base, append(append([]string{}, args...), "--move-owner="+owner)...), 4, "E_OUTPUT_ROOT_MOVE_INVALID")
	}
	for _, override := range []string{"--package-name=org.example.changed", "--model-style=plain_class", "--runtime-package=org.example.runtime", "--lock-hoisted-names=false", "--input-format=json"} {
		fails(t, invoke(t, base, append(append([]string{}, args...), "--move-owner=a", "--move-owner=b", override)...), 1, "E_CONFIG_OVERRIDE")
	}
	for _, run := range [][]string{
		{"generate", "--config", cfg, "--output", dst, "--move-owner=a", "--diagnostic-format=json"},
		{"generate", "--input", filepath.Join(base, "a.json"), "--package-name=org.example.shared", "--output", dst, "--move-output-from", src, "--move-owner=a", "--diagnostic-format=json"},
		{"generate", "--config", cfg, "--move-output-from", src, "--move-owner=a", "--diagnostic-format=json"},
	} {
		fails(t, invoke(t, base, run...), 2, "E_USAGE")
	}
	run := append(append([]string{}, args...), "--move-owner=a", "--move-owner=b")
	// All source inputs remain required, including retained c.
	c := read(t, filepath.Join(base, "c.json"))
	if e := os.Remove(filepath.Join(base, "c.json")); e != nil {
		t.Fatal(e)
	}
	fails(t, invoke(t, base, run...), 4, "E_INPUT_IO")
	put(t, filepath.Join(base, "c.json"), c)
	// Complete source membership, unchanged generation and excluded settings validate.
	put(t, cfg, []byte(strings.Replace(text, "name=\"c\"\nlock_hoisted_names=true", "name=\"c\"\nlock_hoisted_names=false", 1)))
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_ROOT_MOVE_CHANGED")
	put(t, cfg, []byte(text))
	cut := strings.Index(text, "[[specs]]\nname=\"c\"")
	next := strings.Index(text[cut+1:], "[[specs]]") + cut + 1
	put(t, cfg, []byte(text[:cut]+text[next:]))
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_ROOT_MOVE_INVALID")
	put(t, cfg, []byte(strings.Replace(text, "name=\"x\"", "name=\"x\"\nhoist_shared_shapes=false", 1)))
	fails(t, invoke(t, base, run...), 1, "E_OPTIONS")
	put(t, cfg, []byte(text))
	var changed map[string]any
	if e := json.Unmarshal(c, &changed); e != nil {
		t.Fatal(e)
	}
	changed["components"].(map[string]any)["schemas"].(map[string]any)["Added"] = map[string]any{"type": "string"}
	changedBytes, e := json.Marshal(changed)
	if e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(base, "c.json"), changedBytes)
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_ROOT_MOVE_CHANGED")
	put(t, filepath.Join(base, "c.json"), c)
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("refusal changed source")
	}
	m := multipleMoveState(t, src)
	path := m.Owners["a"][0]
	if e := os.MkdirAll(filepath.Dir(filepath.Join(dst, path)), 0700); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(dst, path), read(t, filepath.Join(src, path)))
	targetBefore := snapshot(t, dst)
	fails(t, invoke(t, base, run...), 4, "E_OUTPUT_COLLISION")
	if !reflect.DeepEqual(before, snapshot(t, src)) || !reflect.DeepEqual(targetBefore, snapshot(t, dst)) {
		t.Fatal("collision changed roots")
	}
}

func TestRootMoveMultipleOwnersNativeRuntimePlacement(t *testing.T) {
	for _, declared := range []string{"generated", "target"} {
		for _, selection := range [][]string{{"--move-owner=a", "--move-owner=b"}, {"--spec=a"}} {
			t.Run(declared+strings.Join(selection, ""), func(t *testing.T) {
				base := t.TempDir()
				names := []string{"a", "b", "c"}
				lifecycleConfigInputs(t, base, names)
				cfg := lifecycleConfig(t, base, names, true, false, true)
				result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
				src, dst := filepath.Join(base, "generated"), filepath.Join(base, "target")
				before := snapshot(t, src)
				text := string(read(t, cfg))
				text = strings.Replace(text, "output_dir='generated'", "output_dir='"+declared+"'", 1)
				put(t, cfg, []byte(text))
				for _, name := range names {
					if e := os.Remove(filepath.Join(base, name+".json")); e != nil {
						t.Fatal(e)
					}
				}
				args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst}
				args = append(args, selection...)
				for _, run := range [][]string{args, append(append([]string{}, args...), "--check"), lifecycleValidate(args)} {
					fails(t, invoke(t, base, run...), 1, "E_CONFIG_RUNTIME_ROOTS")
				}
				if !reflect.DeepEqual(before, snapshot(t, src)) {
					t.Fatal("runtime refusal wrote")
				}
				if _, e := os.Lstat(dst); !os.IsNotExist(e) {
					t.Fatal("runtime refusal created target")
				}
			})
		}
	}
	// Shared runtime users a,b may move together when c uses a separate package.
	base := t.TempDir()
	names := []string{"a", "b", "c"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, false, false, true)
	text := strings.Replace(string(read(t, cfg)), "name=\"c\"", "name=\"c\"\nruntime_package=''", 1)
	put(t, cfg, []byte(text))
	result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	before := snapshot(t, filepath.Join(base, "generated"))
	for _, selection := range [][]string{{"--move-owner=a", "--move-owner=b"}, {"--spec=a"}} {
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", "./generated", "--output", "generated/."}
		args = append(args, selection...)
		for _, run := range [][]string{args, append(append([]string{}, args...), "--check"), lifecycleValidate(args)} {
			out := invoke(t, base, run...)
			result(t, out, 0)
			if !bytes.Contains(out.Stdout, []byte(`"identity":true`)) {
				t.Fatal("same-root identity", string(out.Stdout))
			}
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, filepath.Join(base, "generated"))) {
		t.Fatal("identity wrote")
	}
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", "generated", "--output", "target", "--move-owner=b", "--move-owner=a"}
	result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
	result(t, invoke(t, base, args...), 0)
}
