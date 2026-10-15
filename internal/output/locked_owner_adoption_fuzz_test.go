package output

import (
	"path/filepath"
	"reflect"
	"testing"
)

func FuzzLockedOwnerAdoptionPreservesOldState(f *testing.F) {
	for _, seed := range []byte{0, 1, 2, 3, 4, 5, 6} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, mode byte) {
		root, u := lockedAdoptionFixture(t)
		code := ""
		switch mode % 7 {
		case 1:
			u.Generated.Files = u.Generated.Files[1:]
			code = "E_LOCKED_OWNER_ADOPTION_CHANGED"
		case 2:
			u.Generated.NameLock = nil
			code = "E_LOCKED_OWNER_ADOPTION_CHANGED"
		case 3:
			u.Previous.Manifest = append(u.Previous.Manifest, ' ')
			code = "E_LOCKED_OWNER_ADOPTION_STALE"
		case 4:
			put(t, filepath.Join(root, nameLockFile), []byte("modified"))
			code = "E_NAME_LOCK_MODIFIED"
		case 5:
			put(t, filepath.Join(root, ownerNameLockFile(u.Owner)), u.Generated.NameLock)
			code = "E_OUTPUT_COLLISION"
		case 6:
			u.Generated.Files[0].Content += " "
			code = "E_LOCKED_OWNER_ADOPTION_CHANGED"
		}
		before := snapshot(t, root)
		_, e := WriteLockedOwnerAdoption(root, u, true)
		if code != "" {
			errorCode(t, e, code)
		} else if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("read-only changed output")
		}
		if code != "" {
			_, e = WriteLockedOwnerAdoption(root, u, false)
			errorCode(t, e, code)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("failure changed output")
			}
		}
	})
}
