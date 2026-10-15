package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveLegalMixedEmptyV4AndNoncanonicalLockBytes(t *testing.T) {
	for _, kind := range []string{"mixed-v4", "empty-lock-v4", "raw-v3", "raw-v4"} {
		version := 4
		if kind == "raw-v3" {
			version = 3
		}
		src, dst, _ := moveFixture(t, version)
		raw := mustRead(t, filepath.Join(src, manifestName))
		m, e := parseManifestKind(raw, version == 4, true)
		if e != nil {
			t.Fatal(e)
		}
		if kind == "mixed-v4" || kind == "empty-lock-v4" {
			remove := []string{"peer"}
			if kind == "empty-lock-v4" {
				remove = append(remove, "alpha")
			}
			for _, owner := range remove {
				delete(*m.NameLocks, owner)
				if e := os.Remove(filepath.Join(src, ownerNameLockFile(owner))); e != nil {
					t.Fatal(e)
				}
			}
		} else {
			path := nameLockFile
			if version == 4 {
				path = ownerNameLockFile("peer")
			}
			data := append(mustRead(t, filepath.Join(src, path)), '\n', ' ')
			put(t, filepath.Join(src, path), data)
			if version == 3 {
				hash := digest(data)
				m.NameLock = &hash
			} else {
				(*m.NameLocks)["peer"] = digest(data)
			}
		}
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		put(t, filepath.Join(src, manifestName), data)
		s, e := ReadRootMoveState(src)
		if e != nil {
			t.Fatal(e)
		}
		before := snapshot(t, src)
		if changed, e := MoveOutputRoot(src, dst, s, false); e != nil || !changed {
			t.Fatal(changed, e)
		}
		next, e := ReadRootMoveState(dst)
		if e != nil {
			t.Fatal(e)
		}
		expected := s
		expected.root = next.root
		if !reflect.DeepEqual(next, expected) || next.Version != uint32(version) {
			t.Fatal("legal meaning/raw bytes lost")
		}
		after := snapshot(t, dst)
		for path := range s.Entries {
			if !reflect.DeepEqual(before[filepath.FromSlash(path)], after[filepath.FromSlash(path)]) {
				t.Fatal("raw state/attributes changed", path)
			}
		}
	}
}

func TestRootMoveValidPeerRawStateUpdateUnderLocksCAS(t *testing.T) {
	src, dst, s := moveFixture(t, 4)
	var before map[string]snapshotEntry
	_, e := moveOutputRoot(src, dst, s, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			path := filepath.Join(src, ownerNameLockFile("peer"))
			data := append(mustRead(t, path), ' ')
			put(t, path, data)
			m, e := readManifestKind(src, true, true)
			if e != nil {
				t.Fatal(e)
			}
			(*m.NameLocks)["peer"] = digest(data)
			encoded, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(src, manifestName), encoded)
			if _, e := ReadRootMoveState(src); e != nil {
				t.Fatal("competing state not valid", e)
			}
			before = snapshot(t, src)
		}
		return nil
	})
	errorCode(t, e, "E_OUTPUT_ROOT_MOVE_STALE")
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("lost valid peer CAS update")
	}
}
