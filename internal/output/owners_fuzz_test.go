package output

import (
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func FuzzOwnerTransitions(f *testing.F) {
	for _, seed := range [][]byte{{0, 1, 2, 3}, {1, 0, 4, 5}, {2, 2, 2}, {0, 5, 1, 4, 2}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, steps []byte) {
		if len(steps) > 8 {
			t.Skip()
		}
		root := t.TempDir()
		if _, err := WriteOwners(root, OwnerUpdate{Reconcile: true}, false); err != nil {
			t.Fatal(err)
		}
		model := map[string]core.GenerationArtifacts{}
		for _, step := range steps {
			owner := "alpha"
			name := "Alpha"
			if step%2 == 1 {
				owner = "beta"
				name = "Beta"
			}
			u := OwnerUpdate{ActiveOwners: []string{"alpha", "beta"}, Generated: map[string]core.GenerationArtifacts{}}
			switch step % 4 {
			case 0, 1:
				a := ownerArtifacts(name)
				if step&4 != 0 {
					a = core.GenerationArtifacts{}
				}
				u.Generated[owner] = a
				if len(a.Files) == 0 {
					delete(model, owner)
				} else {
					model[owner] = a
				}
			case 2:
				u.ActiveOwners = []string{"alpha"}
				u.Reconcile = true
				delete(model, "beta")
			case 3:
				u.Generated[owner] = model[owner]
				before := snapshot(t, root)
				if changed, e := WriteOwners(root, u, true); e != nil || changed {
					t.Fatal("model-identical check", changed, e)
				}
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("check wrote output")
				}
				continue
			}
			if _, err := WriteOwners(root, u, false); err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{}
			for _, a := range model {
				for _, file := range a.Files {
					expected[file.Path] = file.Content
				}
			}
			m, e := readManifestMode(root, true)
			if e != nil || len(m.Files) != len(expected) {
				t.Fatal(m, e, expected)
			}
			for path, content := range expected {
				actual, err := os.ReadFile(filepath.Join(root, path))
				if err != nil || string(actual) != content {
					t.Fatal("reference model", path, err)
				}
			}
			if c, e := WriteOwners(root, u, true); e != nil || c {
				t.Fatal("postcommit check", c, e)
			}
		}
	})
}
