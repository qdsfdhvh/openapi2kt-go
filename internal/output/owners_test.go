package output

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func ownerArtifacts(name string) core.GenerationArtifacts {
	return artifacts(map[string][]byte{name + ".kt": []byte("public object " + name + "\n"), "Runtime.kt": []byte("public object Runtime\n")})
}
func ownersUpdate() OwnerUpdate {
	return OwnerUpdate{ActiveOwners: []string{"alpha", "beta"}, Generated: map[string]core.GenerationArtifacts{"alpha": ownerArtifacts("Alpha"), "beta": ownerArtifacts("Beta")}}
}
func TestMultiOwnerStructuralControl(t *testing.T) {
	a := ownerArtifacts("Alpha")
	a.Files = append(a.Files, ownerArtifacts("Beta").Files[0])
	if changed, e := Write(t.TempDir(), a, false); e != nil || !changed {
		t.Fatal(changed, e)
	}
}
func TestMultiOwnerPartialAndShared(t *testing.T) {
	root := t.TempDir()
	u := ownersUpdate()
	if c, e := WriteOwners(root, u, false); e != nil || !c {
		t.Fatal(c, e)
	}
	put(t, filepath.Join(root, "Notes.kt"), []byte("synthetic handwritten"))
	before := snapshot(t, root)
	if c, e := WriteOwners(root, u, true); e != nil || c {
		t.Fatal(c, e)
	}
	if c, e := WriteOwners(root, u, false); e != nil || c {
		t.Fatal(c, e)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("no-op/check modified owned files")
	}
	u.Generated = map[string]core.GenerationArtifacts{"alpha": ownerArtifacts("AlphaNew")}
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "Alpha.kt")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	for _, p := range []string{"AlphaNew.kt", "Beta.kt", "Runtime.kt", "Notes.kt"} {
		if _, e := os.Stat(filepath.Join(root, p)); e != nil {
			t.Fatal(p, e)
		}
	}
	u.Generated = map[string]core.GenerationArtifacts{"alpha": {}}
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "Runtime.kt")); e != nil {
		t.Fatal("unselected beta lost shared runtime", e)
	}
	u.Generated = nil
	u.ActiveOwners = []string{"beta"}
	u.Reconcile = true
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	u.ActiveOwners = nil
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{"Beta.kt", "Runtime.kt"} {
		if _, e := os.Stat(filepath.Join(root, p)); !os.IsNotExist(e) {
			t.Fatal("last owner cleanup", p, e)
		}
	}
	if string(readOwnerFile(t, root, "Notes.kt")) != "synthetic handwritten" {
		t.Fatal("deleted handwritten")
	}
	var m struct {
		Version int
		Files   map[string]string
		Owners  map[string][]string
	}
	if e := json.Unmarshal(readOwnerFile(t, root, manifestName), &m); e != nil || m.Version != 2 || len(m.Files) != 0 || m.Owners == nil || len(m.Owners) != 0 {
		t.Fatal(m, e)
	}
}
func readOwnerFile(t *testing.T, root, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(root, path))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestMultiOwnerConflictsAndLegacy(t *testing.T) {
	root := t.TempDir()
	u := ownersUpdate()
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	errorCode(t, func() *core.Diagnostic { _, e := Write(root, ownerArtifacts("Alpha"), false); return e }(), "E_MANIFEST_VERSION")
	u.Generated = map[string]core.GenerationArtifacts{"alpha": artifacts(map[string][]byte{"Runtime.kt": []byte("different")})}
	for _, check := range []bool{false, true} {
		_, e := WriteOwners(root, u, check)
		errorCode(t, e, "E_OUTPUT_COLLISION")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("conflicting shared write changed output")
		}
	}
	legacy := t.TempDir()
	if _, e := Write(legacy, ownerArtifacts("Alpha"), false); e != nil {
		t.Fatal(e)
	}
	before = snapshot(t, legacy)
	_, e := WriteOwners(legacy, ownersUpdate(), false)
	errorCode(t, e, "E_MANIFEST_OWNER_MIGRATION")
	if !reflect.DeepEqual(before, snapshot(t, legacy)) {
		t.Fatal("adopted legacy owner without authority")
	}
}
