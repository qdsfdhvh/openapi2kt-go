package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNamedHierarchyReferenceChains(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-reference-chains.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	for _, n := range []int{1, 45, 46, 47, 48} {
		f.Add(hierarchyInternalChainInput(n, true))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, validation := range []bool{false, true} {
			r := request(b, "Dto")
			r.Options.Validation = validation
			a, e := core.GenerateWithDiagnostics(r)
			c, e2 := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(e) != fmt.Sprint(e2) {
				t.Fatal("nondeterministic chain")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial failure")
			}
		}
	})
}
