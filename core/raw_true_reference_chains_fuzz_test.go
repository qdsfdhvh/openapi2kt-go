package core_test

import (
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"testing"
)

func FuzzRawTrueReferenceChains(f *testing.F) {
	b, err := os.ReadFile("../tests/fixtures/http/raw-true-reference-chains.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Add(schemas(`{` + rawTrueChainCatalog(`true`) + `}`))
	f.Add(schemas(`{` + rawTrueChainCatalog(`{"allOf":[{"type":"object"}]}`) + `}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(b, "Dto")
			r.Options.Validation = checked
			a, ea := core.GenerateWithDiagnostics(r)
			c, ec := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(ea) != fmt.Sprint(ec) {
				t.Fatal("nondeterministic raw/true source graph")
			}
			if ea != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("failed raw/true source returned artifacts/warnings")
			}
		}
	})
}
