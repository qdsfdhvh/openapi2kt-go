package output

import (
	"openapi-kt/core"
	"os"
	"path/filepath"
	"testing"
)

func TestNameLockMetadataNeverSilentlyDiscarded(t *testing.T) {
	for _, check := range []bool{false, true} {
		for _, state := range [][]byte{[]byte{}, []byte(`{"version":1}`)} {
			root := filepath.Join(t.TempDir(), "absent")
			artifact := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "models/A.kt", Content: "old"}}, NameLock: state}
			_, e := Write(root, artifact, check)
			if e == nil || e.Code != "E_NAME_LOCK_OUTPUT_UNSUPPORTED" {
				t.Fatal("direct drops state", e)
			}
			update := OwnerUpdate{ActiveOwners: []string{"a"}, Generated: map[string]core.GenerationArtifacts{"a": artifact}}
			_, e = WriteOwners(root, update, check)
			if e == nil || e.Code != "E_NAME_LOCK_OUTPUT_UNSUPPORTED" {
				t.Fatal("named drops state", e)
			}
			_, e = WriteOwnersBatch(map[string]OwnerUpdate{root: update}, check)
			if e == nil || e.Code != "E_NAME_LOCK_OUTPUT_UNSUPPORTED" {
				t.Fatal("batch drops state", e)
			}
			if _, e := os.Stat(root); !os.IsNotExist(e) {
				t.Fatal("unsupported metadata created output", e)
			}
		}
	}
}
