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

func legacyOwnerRoot(t testing.TB) (string, OwnerUpdate) {
	t.Helper()
	root := t.TempDir()
	a := ownerArtifacts("Alpha")
	if _, e := Write(root, a, false); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
	return root, OwnerUpdate{ActiveOwners: []string{"alpha", "beta"}, Generated: map[string]core.GenerationArtifacts{"alpha": a}, AdoptLegacy: "alpha"}
}
func TestAdoptionMetadataOnlyAndDefaultRefusal(t *testing.T) {
	root, u := legacyOwnerRoot(t)
	before := snapshot(t, root)
	plain := u
	plain.AdoptLegacy = ""
	_, e := WriteOwners(root, plain, false)
	errorCode(t, e, "E_MANIFEST_OWNER_MIGRATION")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("implicit migration")
	}
	p, e := makeOwnersPlan(root, u)
	if e != nil || !p.Changed || len(p.Writes) != 0 || len(p.Removals) != 0 || p.Manifest.Version != 2 {
		t.Fatal(p, e)
	}
	if c, e := WriteOwners(root, u, true); e != nil || !c {
		t.Fatal(c, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("adoption check wrote")
	}
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	after := snapshot(t, root)
	for path, old := range before {
		if path != manifestName && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("adoption changed files/coordination", path)
		}
	}
	for _, check := range []bool{false, true} {
		if c, e := WriteOwners(root, u, check); e != nil || c {
			t.Fatal(c, e)
		}
	}
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Fatal("adopted noop wrote")
	}
}
func TestAdoptionStrictMismatchAndInputShape(t *testing.T) {
	for _, kind := range []string{"content", "path", "missing", "modified", "wrongowner", "multiowner", "unowned", "version", "corrupt", "unsafe"} {
		t.Run(kind, func(t *testing.T) {
			root, u := legacyOwnerRoot(t)
			want := "E_MANIFEST_OWNER_MISMATCH"
			switch kind {
			case "content":
				a := u.Generated["alpha"]
				a.Files[0].Content = "different"
				u.Generated["alpha"] = a
			case "path":
				a := u.Generated["alpha"]
				a.Files[0].Path = "Other.kt"
				u.Generated["alpha"] = a
			case "missing":
				if e := os.Remove(filepath.Join(root, "Alpha.kt")); e != nil {
					t.Fatal(e)
				}
				want = "E_OUTPUT_MISSING"
			case "modified":
				put(t, filepath.Join(root, "Alpha.kt"), []byte("changed"))
				want = "E_OUTPUT_MODIFIED"
			case "wrongowner":
				u.AdoptLegacy = "beta"
				want = "E_OWNER_UPDATE_INVALID"
			case "multiowner":
				u.Generated["beta"] = ownerArtifacts("Beta")
				want = "E_OWNER_UPDATE_INVALID"
			case "unowned":
				put(t, filepath.Join(root, "Beta.kt"), []byte("handwritten"))
				u.Generated["alpha"] = ownerArtifacts("Beta")
			case "version":
				put(t, filepath.Join(root, manifestName), []byte(`{"version":3,"files":{}}`))
				want = "E_MANIFEST_VERSION"
			case "corrupt":
				put(t, filepath.Join(root, manifestName), []byte(`{"version":1,"files":{},"unknown":true}`))
				want = "E_MANIFEST_INVALID"
			case "unsafe":
				a := u.Generated["alpha"]
				a.Files[0].Path = "../Unsafe.kt"
				u.Generated["alpha"] = a
				want = "E_OUTPUT_PATH"
			}
			before := snapshot(t, root)
			for _, check := range []bool{false, true} {
				_, e := WriteOwners(root, u, check)
				errorCode(t, e, want)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("failed adoption changed output")
				}
			}
		})
	}
}
func TestAdoptionRollbackPreservesOriginalV1(t *testing.T) {
	for _, stagedFailure := range []bool{false, true} {
		root, u := legacyOwnerRoot(t)
		before := snapshot(t, root)
		p, e := makeOwnersPlan(root, u)
		if e != nil {
			t.Fatal(e)
		}
		err := commit(root, p, func(index int, stage string) error {
			if index != 0 {
				t.Fatal("metadata-only adoption added file operations")
			}
			if stagedFailure {
				var j journal
				if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
					t.Fatal(e)
				}
				if !j.Operations[0].HadOld || j.Operations[0].Path != manifestName {
					t.Fatal(j)
				}
				return os.Remove(filepath.Join(stage, *j.Operations[0].Staged))
			}
			return errors.New("adoption boundary failure")
		})
		errorCode(t, err, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("rollback did not restore original v1 bytes/mtime")
		}
	}
}
