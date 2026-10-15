package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMakePlanFromRechecksParentsBetweenPasses(t *testing.T) {
	for _, initiallyPresent := range []bool{true, false} {
		name := "missing ancestor becomes directory"
		if initiallyPresent {
			name = "directory replaced by file"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "pkg")
			if initiallyPresent {
				if err := os.Mkdir(parent, 0700); err != nil {
					t.Fatal(err)
				}
			}
			desired := map[string][]byte{"pkg/A.kt": []byte("a"), "pkg/nested/B.kt": []byte("b")}
			before := snapshot(t, root)
			p, err := makePlanFrom(root, desired, nil, manifest{Version: 1, Files: map[string]string{}})
			if err != nil || p == nil || !p.Changed || len(p.Writes) != len(desired) {
				t.Fatal("initial plan", p, err)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("planning created missing directories or files")
			}
			if initiallyPresent {
				if err := os.Remove(parent); err != nil {
					t.Fatal(err)
				}
				put(t, parent, []byte("obstruction"))
			} else {
				if err := os.Mkdir(parent, 0700); err != nil {
					t.Fatal(err)
				}
				put(t, filepath.Join(parent, "nested"), []byte("obstruction"))
			}
			before = snapshot(t, root)
			_, err = makePlanFrom(root, desired, nil, manifest{Version: 1, Files: map[string]string{}})
			errorCode(t, err, "E_OUTPUT_PATH")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("failed replan mutated output")
			}
		})
	}
}

func TestMakePlanFromChecksLaterSiblingParents(t *testing.T) {
	for _, phase := range []string{"owned", "desired"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "pkg", "a"), 0700); err != nil {
				t.Fatal(err)
			}
			a, b := []byte("a"), []byte("b")
			put(t, filepath.Join(root, "pkg", "a", "A.kt"), a)
			put(t, filepath.Join(root, "pkg", "b"), []byte("obstruction"))
			previous := &manifest{Version: 1, Files: map[string]string{"pkg/a/A.kt": digest(a)}}
			if phase == "owned" {
				previous.Files["pkg/b/B.kt"] = digest(b)
			}
			desired := map[string][]byte{"pkg/a/A.kt": a, "pkg/b/B.kt": b}
			before := snapshot(t, root)
			_, err := makePlanFrom(root, desired, previous, manifest{Version: 1, Files: map[string]string{}})
			errorCode(t, err, "E_OUTPUT_PATH")
			if err.Message != "An output parent is not a real directory." {
				t.Fatal("skipped invalid sibling parent", err)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("planning mutated output")
			}
		})
	}
}

func TestMakePlanFromChecksEveryFileAfterSharedParents(t *testing.T) {
	for _, tc := range []struct {
		name      string
		owned     bool
		directory bool
		code      string
	}{
		{"owned directory", true, true, "E_OUTPUT_PATH"},
		{"modified owned file", true, false, "E_OUTPUT_MODIFIED"},
		{"unowned directory", false, true, "E_OUTPUT_PATH"},
		{"unowned identical file", false, false, "E_OUTPUT_COLLISION"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "pkg")
			if err := os.Mkdir(parent, 0700); err != nil {
				t.Fatal(err)
			}
			a, b := []byte("a"), []byte("b")
			put(t, filepath.Join(parent, "A.kt"), a)
			if tc.directory {
				if err := os.Mkdir(filepath.Join(parent, "B.kt"), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				put(t, filepath.Join(parent, "B.kt"), b)
			}
			previous := &manifest{Version: 1, Files: map[string]string{"pkg/A.kt": digest(a)}}
			if tc.owned {
				previous.Files["pkg/B.kt"] = digest([]byte("old b"))
			}
			// The earlier per-file error must precede this later parent error.
			put(t, filepath.Join(parent, "z"), []byte("obstruction"))
			desired := map[string][]byte{"pkg/A.kt": a, "pkg/B.kt": b, "pkg/z/C.kt": []byte("c")}
			before := snapshot(t, root)
			_, err := makePlanFrom(root, desired, previous, manifest{Version: 1, Files: map[string]string{}})
			errorCode(t, err, tc.code)
			if tc.directory && err.Message != "Output entries must be regular files, not directories or symbolic links." {
				t.Fatal("parent inspection changed first diagnostic", err)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("planning mutated output")
			}
		})
	}
}
