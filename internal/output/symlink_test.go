//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package output

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootParentDestinationManifestAndWriterLockSymlinkGuards(t *testing.T) {
	temp := t.TempDir()
	outside := filepath.Join(temp, "outside")
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(outside, "manual.kt"), []byte("keep"))
	root := filepath.Join(temp, "output")
	desired := artifacts(map[string][]byte{"pkg/A.kt": []byte("generated")})
	link := func(source, dest string) {
		t.Helper()
		if err := os.Symlink(source, dest); err != nil {
			t.Fatal(err)
		}
	}
	link(outside, root)
	for _, check := range []bool{true, false} {
		_, err := Write(root, desired, check)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	link(outside, filepath.Join(root, "pkg"))
	for _, check := range []bool{true, false} {
		_, err := Write(root, desired, check)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	if err := os.Remove(filepath.Join(root, "pkg")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "pkg"), 0700); err != nil {
		t.Fatal(err)
	}
	link(filepath.Join(outside, "manual.kt"), filepath.Join(root, "pkg", "A.kt"))
	for _, check := range []bool{true, false} {
		_, err := Write(root, desired, check)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	if err := os.Remove(filepath.Join(root, "pkg", "A.kt")); err != nil {
		t.Fatal(err)
	}
	link(filepath.Join(outside, "manual.kt"), filepath.Join(root, manifestName))
	for _, check := range []bool{true, false} {
		_, err := Write(root, desired, check)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	if err := os.Remove(filepath.Join(root, manifestName)); err != nil {
		t.Fatal(err)
	}
	link(filepath.Join(outside, "manual.kt"), filepath.Join(root, lockName))
	_, err := Write(root, desired, false)
	errorCode(t, err, "E_OUTPUT_PATH")
	before := snapshot(t, root)
	if changed, err := Write(root, desired, true); err != nil || !changed {
		t.Fatal("check should not open/inspect writer lock", changed, err)
	}
	if !reflect.DeepEqual(snapshot(t, root), before) {
		t.Fatal("check mutated symbolic lock")
	}
	if string(mustRead(t, filepath.Join(outside, "manual.kt"))) != "keep" {
		t.Fatal("outside target changed")
	}
	entries, errOS := os.ReadDir(outside)
	if errOS != nil || len(entries) != 1 {
		t.Fatal("escaped output", entries, errOS)
	}
}

func TestMakePlanFromRechecksParentReplacedBySymlink(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "pkg")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if err := inspectParents(root, "pkg/A.kt"); err != nil {
		t.Fatal(err)
	}
	desired := map[string][]byte{"pkg/A.kt": []byte("a"), "pkg/nested/B.kt": []byte("b")}
	if _, err := makePlanFrom(root, desired, nil, manifest{Version: 1, Files: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, parent); err != nil {
		t.Fatal(err)
	}
	_, err := makePlanFrom(root, desired, nil, manifest{Version: 1, Files: map[string]string{}})
	errorCode(t, err, "E_OUTPUT_PATH")
	// Existing callers also need fresh inspection on every call.
	errorCode(t, inspectParents(root, "pkg/A.kt"), "E_OUTPUT_PATH")
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatal("planning changed symlink target", entries, err)
	}
}

func TestAncestorsAboveExplicitRootUseNormalFilesystemResolution(t *testing.T) {
	temp := t.TempDir()
	real := filepath.Join(temp, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(temp, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(alias, "generated")
	files := artifacts(map[string][]byte{"A.kt": []byte("a")})
	if changed, err := Write(root, files, false); err != nil || !changed {
		t.Fatal(changed, err)
	}
	if string(mustRead(t, filepath.Join(real, "generated", "A.kt"))) != "a" {
		t.Fatal("root not resolved")
	}
}
