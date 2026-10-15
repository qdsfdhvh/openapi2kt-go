package output

import (
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzExactLegacyAdoption(f *testing.F) {
	for _, seed := range []string{"", "synthetic", "unicode-字", "line\nnext"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, content string) {
		if len(content) > 256 {
			t.Skip()
		}
		root := t.TempDir()
		a := artifacts(map[string][]byte{"Alpha.kt": []byte(content)})
		if _, e := Write(root, a, false); e != nil {
			t.Fatal(e)
		}
		u := OwnerUpdate{ActiveOwners: []string{"alpha"}, Generated: map[string]core.GenerationArtifacts{"alpha": a}, AdoptLegacy: "alpha"}
		before := snapshot(t, root)
		if c, e := WriteOwners(root, u, true); e != nil || !c {
			t.Fatal(c, e)
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("migration check changed old tree")
		}
		if _, e := WriteOwners(root, u, false); e != nil {
			t.Fatal(e)
		}
		after := snapshot(t, root)
		if !reflect.DeepEqual(before["Alpha.kt"], after["Alpha.kt"]) {
			t.Fatal("metadata adoption rewrote source")
		}
		if c, e := WriteOwners(root, u, false); e != nil || c {
			t.Fatal(c, e)
		}
		if !reflect.DeepEqual(after, snapshot(t, root)) {
			t.Fatal("v2 idempotency")
		}
	})
}
