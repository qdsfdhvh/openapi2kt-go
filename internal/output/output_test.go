package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"openapi-kt/core"
)

type snapshotEntry struct {
	Directory bool
	Data      string
	Modified  time.Time
	Mode      os.FileMode
	Size      int64
}

func snapshot(t testing.TB, root string) map[string]snapshotEntry {
	t.Helper()
	entries, err := snapshotTree(root, os.ReadFile)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func snapshotTree(root string, readFile func(string) ([]byte, error)) (map[string]snapshotEntry, error) {
	entries := map[string]snapshotEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			entries[relative] = snapshotEntry{Directory: true}
			return nil
		}
		meta, err := entry.Info()
		if err != nil {
			return err
		}
		var data []byte
		// An empty regular file's metadata determines its bytes. Reading even
		// at EOF can hit Windows' mandatory byte-zero lock on the sidecar.
		// Retain its entry and all metadata; nonempty files and symlinks still
		// require a successful read and must never be silently omitted.
		if !meta.Mode().IsRegular() || meta.Size() != 0 {
			data, err = readFile(path)
			if err != nil {
				return err
			}
		}
		entries[relative] = snapshotEntry{Data: string(data), Modified: meta.ModTime(), Mode: meta.Mode(), Size: meta.Size()}
		return nil
	})
	return entries, err
}
func put(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func planned(t testing.TB, root string, files map[string][]byte) *plan {
	t.Helper()
	p, err := makePlan(root, files)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func seeded(t testing.TB, root string, files map[string][]byte) {
	t.Helper()
	if err := commit(root, planned(t, root, files), nil); err != nil {
		t.Fatal(err)
	}
}
func errorCode(t testing.TB, err *core.Diagnostic, code string) {
	t.Helper()
	if err == nil || err.Code != code || err.Pointer != nil {
		t.Fatalf("want %s, got %v", code, err)
	}
}
func artifacts(files map[string][]byte) core.GenerationArtifacts {
	out := core.GenerationArtifacts{}
	for _, name := range sortedKeys(files) {
		out.Files = append(out.Files, core.GeneratedFile{Path: name, Content: string(files[name])})
	}
	return out
}

func TestEachCommitBoundaryRestoresCompletePreviousTree(t *testing.T) {
	for failureAt := range 4 {
		t.Run(string(rune('0'+failureAt)), func(t *testing.T) {
			root := t.TempDir()
			seeded(t, root, map[string][]byte{"a/A.kt": []byte("old-a"), "a/B.kt": []byte("old-b")})
			put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
			before := snapshot(t, root)
			next := planned(t, root, map[string][]byte{"a/A.kt": []byte("new-a"), "b/nested/C.kt": []byte("new-c")})
			err := commit(root, next, func(index int, stage string) error {
				if index == failureAt {
					return errors.New("injected boundary")
				}
				return nil
			})
			errorCode(t, err, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(snapshot(t, root), before) {
				t.Fatal("rollback changed tree/bytes/mtimes/modes")
			}
		})
	}
}
func TestFailureAfterOldFileAndManifestBackupRestores(t *testing.T) {
	for _, failureAt := range []int{0, 1} {
		t.Run(string(rune('0'+failureAt)), func(t *testing.T) {
			root := t.TempDir()
			seeded(t, root, map[string][]byte{"A.kt": []byte("old")})
			before := snapshot(t, root)
			err := commit(root, planned(t, root, map[string][]byte{"A.kt": []byte("new")}), func(index int, stage string) error {
				if index == failureAt {
					var j journal
					if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); err != nil {
						t.Fatal(err)
					}
					if !j.Operations[index].HadOld {
						t.Fatal("journal missing old ownership")
					}
					return os.Remove(filepath.Join(stage, *j.Operations[index].Staged))
				}
				return nil
			})
			errorCode(t, err, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(snapshot(t, root), before) {
				t.Fatal("backup-move rollback")
			}
		})
	}
}
func mustRead(t testing.TB, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestFailedRollbackRetainsRecoveryEvidenceAndBlocksBothModes(t *testing.T) {
	root := t.TempDir()
	seeded(t, root, map[string][]byte{"A.kt": []byte("old-a")})
	previous := mustRead(t, filepath.Join(root, manifestName))
	err := commit(root, planned(t, root, map[string][]byte{"A.kt": []byte("new-a")}), func(index int, stage string) error {
		if index == 1 {
			if err := os.Remove(filepath.Join(root, "A.kt")); err != nil {
				return err
			}
			if err := os.Mkdir(filepath.Join(root, "A.kt"), 0700); err != nil {
				return err
			}
			return errors.New("rollback obstruction")
		}
		return nil
	})
	errorCode(t, err, "E_RECOVERY_REQUIRED")
	if !bytes.Equal(mustRead(t, filepath.Join(root, manifestName)), previous) {
		t.Fatal("old manifest lost")
	}
	entries, errOS := os.ReadDir(root)
	if errOS != nil {
		t.Fatal(errOS)
	}
	stage := ""
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagePrefix) {
			stage = filepath.Join(root, entry.Name())
		}
	}
	if stage == "" {
		t.Fatal("missing retained stage")
	}
	if string(mustRead(t, filepath.Join(stage, "old-0"))) != "old-a" {
		t.Fatal("backup lost")
	}
	var j journal
	if err := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); err != nil {
		t.Fatal(err)
	}
	if j.Version != 1 || j.Operations[0].Path != "A.kt" || j.Operations[0].Backup != "old-0" || !j.Operations[0].HadOld {
		t.Fatal(j)
	}
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		_, err := Write(root, artifacts(map[string][]byte{"A.kt": []byte("new-a")}), check)
		errorCode(t, err, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(snapshot(t, root), before) {
			t.Fatal("recovery evidence mutated")
		}
	}
}

func TestManifestStrictSlotsUnicodePathsAndHashValidation(t *testing.T) {
	root := t.TempDir()
	hash := strings.Repeat("0", 64)
	for _, tc := range []struct{ data, code string }{
		{"not-json", "E_MANIFEST_INVALID"}, {`{"version":9,"files":{}}`, "E_MANIFEST_VERSION"},
		{`{"version":1.0,"files":{}}`, "E_MANIFEST_INVALID"}, {`{"version":-0,"files":{}}`, "E_MANIFEST_INVALID"},
		{`{"version":4294967296,"files":{}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"../outside.kt":"` + hash + `"}}`, "E_OUTPUT_PATH"},
		{`{"version":1,"files":{"/absolute.kt":"` + hash + `"}}`, "E_OUTPUT_PATH"},
		{`{"version":1,"files":{"a\\b.kt":"` + hash + `"}}`, "E_OUTPUT_PATH"},
		{`{"version":1,"files":{"A.kt":"` + hash + `","a.kt":"` + hash + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":"` + strings.Repeat("g", 64) + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":"` + hash[:63] + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{},"unknown":true}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":"` + hash + `","A.kt":"` + hash + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"version":1,"files":{}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"\ud800.kt":"` + hash + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":false}}`, "E_MANIFEST_INVALID"},
		{`{"version":9,"files":{"A.kt":false}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{},"x":"` + string([]byte{255}) + `"}`, "E_MANIFEST_INVALID"},
	} {
		put(t, filepath.Join(root, manifestName), []byte(tc.data))
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			_, err := Write(root, artifacts(map[string][]byte{"A.kt": []byte("data")}), check)
			errorCode(t, err, tc.code)
			if !reflect.DeepEqual(snapshot(t, root), before) {
				t.Fatal("invalid manifest mutated output")
			}
		}
	}
}

func TestMissingOwnedEmptyFilesNoopStaleAndUnownedOwnership(t *testing.T) {
	root := t.TempDir()
	desired := artifacts(map[string][]byte{"a/A.kt": {}, "a/B.kt": []byte("b")})
	changed, err := Write(root, desired, false)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	put(t, filepath.Join(root, "Notes.kt"), []byte("unowned"))
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		changed, err := Write(root, desired, check)
		if err != nil || changed {
			t.Fatal("no-op", changed, err)
		}
		if !reflect.DeepEqual(snapshot(t, root), before) {
			t.Fatal("no-op bytes/mtimes changed")
		}
	}
	if err := os.Remove(filepath.Join(root, "a", "A.kt")); err != nil {
		t.Fatal(err)
	}
	changed, err = Write(root, desired, true)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	if _, err := Write(root, desired, false); err != nil {
		t.Fatal(err)
	}
	next := artifacts(map[string][]byte{"new/C.kt": []byte("c")})
	if _, err := Write(root, next, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "B.kt")); !os.IsNotExist(err) {
		t.Fatal("stale file survived")
	}
	if string(mustRead(t, filepath.Join(root, "Notes.kt"))) != "unowned" {
		t.Fatal("unowned file deleted")
	}
	if err := os.Remove(filepath.Join(root, manifestName)); err != nil {
		t.Fatal(err)
	}
	before = snapshot(t, root)
	for _, check := range []bool{true, false} {
		_, err := Write(root, next, check)
		errorCode(t, err, "E_OUTPUT_COLLISION")
		if !reflect.DeepEqual(snapshot(t, root), before) {
			t.Fatal("unowned file adopted")
		}
	}
}

func TestReadonlyCheckAbsentAndBusyLocksAndSafeDifferences(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	desired := artifacts(map[string][]byte{"A.kt": []byte("a")})
	changed, err := Write(root, desired, true)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("check created root")
	}
	if _, err := Write(root, desired, false); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, lockName)
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	meta, _ := os.Stat(root)
	if changed, err := Write(root, desired, true); err != nil || changed {
		t.Fatal(changed, err)
	}
	after, _ := os.Stat(root)
	if !reflect.DeepEqual(snapshot(t, root), before) || !meta.ModTime().Equal(after.ModTime()) {
		t.Fatal("check mutated root/entries")
	}
	lock, errOS := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0600)
	if errOS != nil {
		t.Fatal(errOS)
	}
	defer lock.Close()
	if err := tryLock(lock); err != nil {
		t.Fatal(err)
	}
	defer unlock(lock)
	before = snapshot(t, root)
	for _, files := range []core.GenerationArtifacts{desired, artifacts(map[string][]byte{"A.kt": []byte("different")})} {
		if _, err := Write(root, files, true); err != nil {
			t.Fatal("check required writer lock", err)
		}
	}
	_, err = Write(root, desired, false)
	errorCode(t, err, "E_OUTPUT_BUSY")
	if !reflect.DeepEqual(snapshot(t, root), before) {
		t.Fatal("busy/check mutated entries")
	}
}

func TestModifiedOwnedSafetyIncludesStaleRemovalAndBothModes(t *testing.T) {
	root := t.TempDir()
	desired := artifacts(map[string][]byte{"A.kt": []byte("a"), "B.kt": []byte("b")})
	if _, err := Write(root, desired, false); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(root, "A.kt"), []byte("manual"))
	before := snapshot(t, root)
	for _, check := range []bool{true, false} {
		for _, files := range []core.GenerationArtifacts{desired, artifacts(map[string][]byte{"B.kt": []byte("b")})} {
			_, err := Write(root, files, check)
			errorCode(t, err, "E_OUTPUT_MODIFIED")
			if !reflect.DeepEqual(snapshot(t, root), before) {
				t.Fatal("manual edits changed")
			}
		}
	}
}

func TestRootAndGeneratedPathsFailBeforeMutation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	for _, path := range []string{"../A.kt", "/A.kt", "a\\A.kt", "a:A.kt", "a//A.kt", "a/./A.kt", ".hidden/A.kt", "A.txt"} {
		_, err := Write(root, artifacts(map[string][]byte{path: []byte("a")}), false)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	_, err := Write(root, core.GenerationArtifacts{}, false)
	errorCode(t, err, "E_OUTPUT_EMPTY")
	_, err = Write(root, core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "A.kt", Content: "a"}, {Path: "a.kt", Content: "b"}}}, false)
	errorCode(t, err, "E_OUTPUT_COLLISION")
	for _, root := range []string{"", root + string(filepath.Separator) + ".." + string(filepath.Separator) + "other"} {
		_, err := Write(root, artifacts(map[string][]byte{"A.kt": []byte("a")}), false)
		errorCode(t, err, "E_OUTPUT_PATH")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("failed path created root")
	}
}

func TestExistingFileMetadataLimit(t *testing.T) {
	root := t.TempDir()
	file, err := os.Create(filepath.Join(root, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	for _, check := range []bool{true, false} {
		_, err := Write(root, artifacts(map[string][]byte{"A.kt": []byte("a")}), check)
		errorCode(t, err, "E_OUTPUT_LIMIT")
	}
}
