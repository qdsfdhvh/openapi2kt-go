package output

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestSnapshotEmptyLockedFileRetainsAllState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, lockName)
	put(t, path, nil)
	put(t, filepath.Join(root, "A.kt"), []byte("synthetic"))
	before := snapshot(t, root)
	denied := errors.New("synthetic mandatory byte lock")
	read := func(name string) ([]byte, error) {
		if name == path {
			return nil, denied
		}
		return os.ReadFile(name)
	}
	locked, err := snapshotTree(root, read)
	if err != nil {
		t.Fatalf("empty regular file metadata must determine its bytes without a locked read: %v", err)
	}
	if !reflect.DeepEqual(before, locked) {
		t.Fatal("snapshot lost an entry, bytes, mode or modification time")
	}
	put(t, path, []byte("nonempty lock sentinel"))
	if _, err := snapshotTree(root, read); !errors.Is(err, denied) {
		t.Fatal("a nonempty unreadable file must still fail, not be silently omitted")
	}
}

func TestAllOutputAdaptersRejectOriginalNonportablePaths(t *testing.T) {
	locked := singleGenerated(t, singleLockRequest(t, core.DataClass, true))
	for _, path := range []string{`nested\A.kt`, `..\Outside.kt`, `C:\Outside.kt`, "/Outside.kt", "a:A.kt", "a//A.kt", "a/./A.kt"} {
		for _, adapter := range []string{"single", "owners", "name-lock", "owner-name-lock"} {
			for _, check := range []bool{false, true} {
				t.Run(adapter+"/"+path+"/"+map[bool]string{false: "write", true: "check"}[check], func(t *testing.T) {
					root := t.TempDir()
					before := snapshot(t, root)
					a := artifacts(map[string][]byte{path: []byte("synthetic")})
					var err *core.Diagnostic
					switch adapter {
					case "single":
						_, err = Write(root, a, check)
					case "owners":
						_, err = WriteOwners(root, OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": a}}, check)
					case "name-lock":
						a.NameLock = locked.NameLock
						_, err = WriteNameLocked(root, a, nil, check)
					case "owner-name-lock":
						a.NameLock = locked.NameLock
						_, err = WriteNameLockedOwners(root, NameLockedOwnerUpdate{OwnerUpdate: OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": a}}, PreviousNameLocks: map[string][]byte{"alpha": nil}}, check)
					}
					errorCode(t, err, "E_OUTPUT_PATH")
					if !reflect.DeepEqual(before, snapshot(t, root)) {
						t.Fatal("invalid input changed output entries")
					}
				})
			}
		}
	}
}
