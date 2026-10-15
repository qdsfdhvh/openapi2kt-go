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

func TestMultiOwnerEveryCommitBoundaryRollback(t *testing.T) {
	for _, backupFailure := range []bool{false, true} {
		for failAt := range 4 {
			root := t.TempDir()
			u := ownersUpdate()
			if _, e := WriteOwners(root, u, false); e != nil {
				t.Fatal(e)
			}
			put(t, filepath.Join(root, "Notes.kt"), []byte("handwritten"))
			before := snapshot(t, root)
			u.Generated = map[string]core.GenerationArtifacts{"alpha": artifacts(map[string][]byte{"new/nested/AlphaNew.kt": []byte("new"), "new/Other.kt": []byte("other"), "Runtime.kt": []byte("public object Runtime\n"), "Beta.kt": []byte("public object Beta\n")})}
			p, e := makeOwnersPlan(root, u)
			if e != nil {
				t.Fatal(e)
			}
			err := commit(root, p, func(index int, stage string) error {
				if index != failAt {
					return nil
				}
				if backupFailure {
					var j journal
					if e := json.Unmarshal(mustRead(t, filepath.Join(stage, "transaction.json")), &j); e != nil {
						t.Fatal(e)
					}
					if staged := j.Operations[index].Staged; staged != nil {
						return os.Remove(filepath.Join(stage, *staged))
					}
				}
				return errors.New("injected owner boundary")
			})
			errorCode(t, err, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("owner rollback changed tree/metadata", backupFailure, failAt)
			}
		}
	}
}
func TestMultiOwnerReplanRetainsInterveningOwner(t *testing.T) {
	root := t.TempDir()
	u := ownersUpdate()
	if _, e := WriteOwners(root, u, false); e != nil {
		t.Fatal(e)
	}
	u.Generated = map[string]core.GenerationArtifacts{"alpha": ownerArtifacts("AlphaNew")}
	calls := 0
	_, err := writePlanned(root, false, func(root string) (*plan, *core.Diagnostic) {
		calls++
		p, e := makeOwnersPlan(root, u)
		if calls == 1 && e == nil {
			other := OwnerUpdate{ActiveOwners: []string{"alpha", "beta", "gamma"}, Generated: map[string]core.GenerationArtifacts{"gamma": ownerArtifacts("Gamma")}}
			if _, err := WriteOwners(root, other, false); err != nil {
				t.Fatal(err)
			}
		}
		return p, e
	})
	if err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	if _, e := os.Stat(filepath.Join(root, "Gamma.kt")); e != nil {
		t.Fatal("lost intervening owner", e)
	}
}
func TestMultiOwnerInvalidAndModifiedPreservation(t *testing.T) {
	root := t.TempDir()
	for _, u := range []OwnerUpdate{{}, {ActiveOwners: []string{"a", "a"}, Reconcile: true}, {ActiveOwners: []string{"a/secret"}, Reconcile: true}, {ActiveOwners: []string{"a"}, Generated: map[string]core.GenerationArtifacts{"b": ownerArtifacts("Beta")}}} {
		if _, err := WriteOwners(root, u, false); err == nil {
			t.Fatal(u)
		}
	}
	if _, e := WriteOwners(root, ownersUpdate(), false); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(root, "Runtime.kt"), []byte("modified"))
	before := snapshot(t, root)
	for _, check := range []bool{false, true} {
		_, e := WriteOwners(root, ownersUpdate(), check)
		errorCode(t, e, "E_OUTPUT_MODIFIED")
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("modified shared output overwritten")
		}
	}
}
func TestMultiOwnerManifestMembershipAndVersion(t *testing.T) {
	for _, kind := range []string{"missingowners", "missingcoverage", "emptyowner", "unknownpath", "duplicatepath", "unordered", "wrongowner", "version", "extra", "duplicatekey"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			if _, e := WriteOwners(root, ownersUpdate(), false); e != nil {
				t.Fatal(e)
			}
			var doc map[string]any
			if e := json.Unmarshal(mustRead(t, filepath.Join(root, manifestName)), &doc); e != nil {
				t.Fatal(e)
			}
			owners := doc["owners"].(map[string]any)
			want := "E_MANIFEST_INVALID"
			switch kind {
			case "missingowners":
				delete(doc, "owners")
			case "missingcoverage":
				delete(owners, "beta")
			case "emptyowner":
				owners["alpha"] = []any{}
			case "unknownpath":
				owners["alpha"] = []any{"Unknown.kt"}
			case "duplicatepath":
				owners["alpha"] = []any{"Alpha.kt", "Alpha.kt"}
			case "unordered":
				owners["alpha"] = []any{"Runtime.kt", "Alpha.kt"}
			case "wrongowner":
				owners["a/secret"] = owners["alpha"]
				delete(owners, "alpha")
			case "version":
				doc["version"] = 3
				delete(doc, "owners")
				want = "E_MANIFEST_VERSION"
			case "extra":
				doc["unknown"] = true
			}
			b, _ := json.Marshal(doc)
			if kind == "duplicatekey" {
				b = []byte(`{"version":2,"version":2,"files":{},"owners":{}}`)
			}
			put(t, filepath.Join(root, manifestName), b)
			before := snapshot(t, root)
			_, err := WriteOwners(root, ownersUpdate(), false)
			errorCode(t, err, want)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("bad manifest was reset")
			}
		})
	}
}
