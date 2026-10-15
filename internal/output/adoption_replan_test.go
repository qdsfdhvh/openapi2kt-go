package output

import (
	"encoding/json"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAdoptionRechecksChangedLegacyUnderLock(t *testing.T) {
	root, u := legacyOwnerRoot(t)
	changed := ownerArtifacts("Alpha")
	changed.Files[0].Content = "new legacy writer"
	var written map[string]snapshotEntry
	_, err := writeOwnersBatch(map[string]OwnerUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			if _, e := Write(root, changed, false); e != nil {
				t.Fatal(e)
			}
			written = snapshot(t, root)
		}
		return nil
	})
	errorCode(t, err, "E_MANIFEST_OWNER_MISMATCH")
	if !reflect.DeepEqual(written, snapshot(t, root)) {
		t.Fatal("stale adoption overwrote intervening legacy writer")
	}
}
func TestAdoptionRechecksInterveningNamedOwner(t *testing.T) {
	root, u := legacyOwnerRoot(t)
	_, err := writeOwnersBatch(map[string]OwnerUpdate{root: u}, false, func(_ int, phase string, _ int, _ string) error {
		if phase == "prelock" {
			g := OwnerUpdate{ActiveOwners: []string{"gamma"}, Generated: map[string]core.GenerationArtifacts{"gamma": ownerArtifacts("Alpha")}, AdoptLegacy: "gamma"}
			if _, e := WriteOwners(root, g, false); e != nil {
				t.Fatal(e)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	if e := json.Unmarshal(mustRead(t, filepath.Join(root, manifestName)), &m); e != nil {
		t.Fatal(e)
	}
	if m.Owners == nil || len(*m.Owners) != 2 || !reflect.DeepEqual((*m.Owners)["alpha"], (*m.Owners)["gamma"]) {
		t.Fatal("locked named replan lost intervening owner", m)
	}
}
