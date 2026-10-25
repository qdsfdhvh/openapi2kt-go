package output

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitLockedAdoptionAggregateSourceBudget(t *testing.T) {
	for _, test := range []struct {
		name       string
		extra      int64
		beforeHash bool
	}{{"boundary", 0, false}, {"overflow", 1, false}, {"overflow-before-hash", 1, true}} {
		t.Run(test.name, func(t *testing.T) {
			root, _ := lockedAdoptionFixture(t)
			m, e := readManifestKind(root, false, true)
			if e != nil {
				t.Fatal(e)
			}
			m.Files = map[string]string{}
			// Sparse synthetic files exercise the real 64 MiB boundary with bounded
			// streaming fixture construction rather than a whole-file buffer.
			for i, name := range []string{"ResourceA.kt", "ResourceB.kt"} {
				file, err := os.Create(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				size := int64(maxAdoptionKotlinBytes / 2)
				if i == 1 {
					size += test.extra
				}
				if err := file.Truncate(size); err != nil {
					t.Fatal(err)
				}
				hash := sha256.New()
				if _, err := io.Copy(hash, file); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				m.Files[name] = hex.EncodeToString(hash.Sum(nil))
			}
			if test.beforeHash {
				// Resource limits must reject the complete inventory before content
				// reads/hashing, even if its first payload would fail its hash check.
				m.Files["ResourceA.kt"] = strings.Repeat("0", 64)
			}
			data, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(root, manifestName), data)
			before, err := os.Stat(filepath.Join(root, manifestName))
			if err != nil {
				t.Fatal(err)
			}
			state, e := ReadSplitLockedAdoptionState(root, []string{"left", "right"})
			if test.extra == 1 {
				errorCode(t, e, "E_OUTPUT_LIMIT")
				t.Logf("old Kotlin total=%d limit=%d diagnostic=%s; before_payload_hash=%t", int64(maxAdoptionKotlinBytes)+test.extra, maxAdoptionKotlinBytes, e.Code, test.beforeHash)
			} else if e != nil || len(state.Files) != 2 {
				t.Fatal("exact aggregate boundary", e)
			} else {
				t.Logf("old Kotlin total=%d limit=%d accepted files=%d", maxAdoptionKotlinBytes, maxAdoptionKotlinBytes, len(state.Files))
			}
			after, err := os.Stat(filepath.Join(root, manifestName))
			if err != nil || !before.ModTime().Equal(after.ModTime()) || !bytes.Equal(data, mustRead(t, filepath.Join(root, manifestName))) {
				t.Fatal("budget read changed metadata", err)
			}
		})
	}
}

func TestSplitAdoptionRetainedCandidateBudgets(t *testing.T) {
	b, err := NewSplitAdoptionBudget(nil)
	if err != nil {
		t.Fatal(err)
	}
	b.kotlinBytes = maxAdoptionKotlinBytes - 1
	a := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "A.kt", Content: "v"}}}
	if err := b.Add("left", a); err != nil {
		t.Fatal("exact candidate budget rejected", err)
	}
	errorCode(t, b.Add("right", a), "E_OUTPUT_LIMIT")
	b, _ = NewSplitAdoptionBudget(nil)
	b.lockBytes = maxOwnerNameLockBytes - 1
	if err := b.Add("left", core.GenerationArtifacts{NameLock: []byte("v")}); err != nil {
		t.Fatal("exact metadata budget rejected", err)
	}
	errorCode(t, b.Add("right", core.GenerationArtifacts{NameLock: []byte("v")}), "E_NAME_LOCK_LIMIT")
	b, _ = NewSplitAdoptionBudget(nil)
	b.manifestBytes = maxFileBytes - (2*len("left") + 160 + len("Empty.kt") + 24)
	empty := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "Empty.kt"}}}
	if err := b.Add("left", empty); err != nil {
		t.Fatal("exact ownership metadata budget rejected", err)
	}
	errorCode(t, b.Add("right", empty), "E_OUTPUT_LIMIT")
}
