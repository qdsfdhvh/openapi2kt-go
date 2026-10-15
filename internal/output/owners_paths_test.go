package output

import (
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMultiOwnerPathGuardsAndMissingRetained(t *testing.T) {
	for _, kind := range []string{"case", "duplicate", "unsafe", "unowned", "missing", "recovery"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			u := ownersUpdate()
			if _, e := WriteOwners(root, u, false); e != nil {
				t.Fatal(e)
			}
			u.Generated = map[string]core.GenerationArtifacts{"alpha": ownerArtifacts("AlphaNew")}
			want := "E_OUTPUT_COLLISION"
			switch kind {
			case "case":
				u.Generated["alpha"] = artifacts(map[string][]byte{"runtime.kt": []byte("public object Runtime\n")})
			case "duplicate":
				a := ownerArtifacts("AlphaNew")
				a.Files = append(a.Files, a.Files[0])
				u.Generated["alpha"] = a
			case "unsafe":
				u.Generated["alpha"] = artifacts(map[string][]byte{"../Outside.kt": []byte("invalid")})
				want = "E_OUTPUT_PATH"
			case "unowned":
				put(t, filepath.Join(root, "AlphaNew.kt"), []byte("handwritten even if byte-identical"))
			case "missing":
				if e := os.Remove(filepath.Join(root, "Beta.kt")); e != nil {
					t.Fatal(e)
				}
				want = "E_OUTPUT_MISSING"
			case "recovery":
				if e := os.Mkdir(filepath.Join(root, stagePrefix+"unfinished"), 0700); e != nil {
					t.Fatal(e)
				}
				want = "E_RECOVERY_REQUIRED"
			}
			before := snapshot(t, root)
			for _, check := range []bool{false, true} {
				_, err := WriteOwners(root, u, check)
				errorCode(t, err, want)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("guard changed tree", kind)
				}
			}
		})
	}
}
