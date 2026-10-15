package output

import (
	"encoding/json"
	"openapi-kt/core"
	"openapi-kt/internal/schema"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSingleNameLockUnsafeOutputAndLostOrModifiedState(t *testing.T) {
	generated := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, kind := range []string{"unowned", "missing", "modified", "oversize", "symlink", "directory", "json-file", "nil-state", "empty-state", "expected-empty"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			a := generated
			var expected []byte
			code := "E_OUTPUT_PATH"
			switch kind {
			case "unowned":
				put(t, filepath.Join(root, nameLockFile), generated.NameLock)
				code = "E_OUTPUT_COLLISION"
			case "json-file":
				a.Files = append(append([]core.GeneratedFile(nil), a.Files...), core.GeneratedFile{Path: nameLockFile, Content: "{}"})
			case "nil-state":
				a.NameLock = nil
				code = "E_NAME_LOCK_OUTPUT_INVALID"
			case "empty-state":
				a.NameLock = []byte{}
				code = "E_NAME_LOCK_INVALID"
			case "expected-empty":
				expected = []byte{}
				code = "E_NAME_LOCK_STALE"
			default:
				if _, e := WriteNameLocked(root, a, nil, false); e != nil {
					t.Fatal(e)
				}
				expected = a.NameLock
				switch kind {
				case "missing":
					if e := os.Remove(filepath.Join(root, nameLockFile)); e != nil {
						t.Fatal(e)
					}
					code = "E_NAME_LOCK_MISSING"
				case "modified":
					put(t, filepath.Join(root, nameLockFile), append(append([]byte(nil), a.NameLock...), ' '))
					code = "E_NAME_LOCK_MODIFIED"
				case "oversize":
					put(t, filepath.Join(root, nameLockFile), []byte(strings.Repeat(" ", schema.MaxNameLockBytes+1)))
					code = "E_OUTPUT_LIMIT"
				case "symlink":
					if e := os.Remove(filepath.Join(root, nameLockFile)); e != nil {
						t.Fatal(e)
					}
					put(t, filepath.Join(root, "sentinel"), a.NameLock)
					if e := os.Symlink("sentinel", filepath.Join(root, nameLockFile)); e != nil {
						t.Skip("symlink unavailable")
					}
				case "directory":
					if e := os.Remove(filepath.Join(root, nameLockFile)); e != nil {
						t.Fatal(e)
					}
					if e := os.Mkdir(filepath.Join(root, nameLockFile), 0700); e != nil {
						t.Fatal(e)
					}
				}
			}
			// Snapshot helper follows regular symlinks for this local safe sentinel only.
			before := snapshot(t, root)
			for _, check := range []bool{true, false} {
				_, e := WriteNameLocked(root, a, expected, check)
				errorCode(t, e, code)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("failure changed old output")
				}
			}
			if kind != "json-file" && kind != "nil-state" && kind != "empty-state" && kind != "expected-empty" {
				_, e := ReadNameLock(root)
				errorCode(t, e, code)
			}
		})
	}
}
func TestSingleNameLockStrictManifestEnvelope(t *testing.T) {
	a := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	hash := digest(a.NameLock)
	for _, c := range []struct{ manifest, code string }{
		{`{"version":3,"files":{}}`, "E_MANIFEST_INVALID"},
		{`{"version":3,"files":{},"name_lock":null}`, "E_MANIFEST_INVALID"},
		{`{"version":3,"files":{},"name_lock":"bad"}`, "E_MANIFEST_INVALID"},
		{`{"version":3,"files":{},"name_lock":"` + hash + `","extra":true}`, "E_MANIFEST_INVALID"},
		{`{"version":3,"version":3,"files":{},"name_lock":"` + hash + `"}`, "E_MANIFEST_INVALID"},
		{`{"version":3,"files":{".openapi-kt-names.json":"` + hash + `"},"name_lock":"` + hash + `"}`, "E_OUTPUT_PATH"},
		{`{"version":3,"files":{"../A.kt":"` + hash + `"},"name_lock":"` + hash + `"}`, "E_OUTPUT_PATH"},
		{`{"version":1,"files":{},"name_lock":"` + hash + `"}`, "E_MANIFEST_INVALID"},
		{`{"version":2,"files":{},"owners":{}}`, "E_MANIFEST_VERSION"},
		{`{"version":4,"files":{}}`, "E_MANIFEST_VERSION"},
	} {
		root := t.TempDir()
		put(t, filepath.Join(root, manifestName), []byte(c.manifest))
		put(t, filepath.Join(root, nameLockFile), a.NameLock)
		before := snapshot(t, root)
		_, e := ReadNameLock(root)
		errorCode(t, e, c.code)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLocked(root, a, nil, check)
			errorCode(t, e, c.code)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("invalid manifest mutated output")
		}
	}
}
func TestSingleNameLockModifiedKotlinAndMalformedOwnedState(t *testing.T) {
	req := singleLockRequest(t, core.DataClass, true)
	a := singleGenerated(t, req)
	for _, kind := range []string{"kt", "invalid", "unsupported"} {
		root := t.TempDir()
		if _, e := WriteNameLocked(root, a, nil, false); e != nil {
			t.Fatal(e)
		}
		code := "E_OUTPUT_MODIFIED"
		if kind == "kt" {
			put(t, filepath.Join(root, a.Files[0].Path), []byte("synthetic manually edited Kotlin"))
		} else {
			state := []byte(`{}`)
			code = "E_NAME_LOCK_INVALID"
			if kind == "unsupported" {
				state = []byte(strings.Replace(string(a.NameLock), `"version":1`, `"version":2`, 1))
				code = "E_NAME_LOCK_VERSION"
			}
			put(t, filepath.Join(root, nameLockFile), state)
			hash := digest(state)
			m := manifest{Version: 3, Files: map[string]string{}, NameLock: &hash}
			for _, f := range a.Files {
				m.Files[f.Path] = digest([]byte(f.Content))
			}
			b, _ := json.Marshal(m)
			put(t, filepath.Join(root, manifestName), b)
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, e := WriteNameLocked(root, a, a.NameLock, check)
			errorCode(t, e, code)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("unsafe tree modified")
		}
	}
}
func FuzzSingleNameLockManifest(f *testing.F) {
	for _, s := range []string{`{"version":1,"files":{}}`, `{"version":3,"files":{},"name_lock":"` + strings.Repeat("0", 64) + `"}`, `{"version":3,"files":{},"name_lock":null}`, `{}`} {
		f.Add(s)
	}
	root := f.TempDir()
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 8192 {
			t.Skip()
		}
		put(t, filepath.Join(root, manifestName), []byte(data))
		m, e := readManifestKind(root, false, true)
		if e == nil {
			if m == nil || m.Version != 1 && m.Version != 3 || m.Version == 3 && (m.NameLock == nil || !validContentHash(*m.NameLock)) {
				t.Fatal("invalid accepted manifest")
			}
			for p := range m.Files {
				if e := relativePath(p); e != nil {
					t.Fatal(e)
				}
			}
		}
	})
}
