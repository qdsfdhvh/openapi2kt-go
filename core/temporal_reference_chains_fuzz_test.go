package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzTemporalReferenceChains(f *testing.F) {
	b, e := os.ReadFile("../tests/fixtures/http/temporal-reference-chains.json")
	if e != nil {
		f.Fatal(e)
	}
	f.Add(b)
	f.Add(schemas(`{"Time":{"type":"string","format":"date-time","nullable":true},"Hops":{"properties":{"time":{"$ref":"#/components/schemas/Time"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/time","nullable":false}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(b, "Dto")
			r.Options.Validation = checked
			a, e := core.GenerateWithDiagnostics(r)
			c, e2 := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, c) || fmt.Sprint(e) != fmt.Sprint(e2) {
				t.Fatal("nondeterministic temporal chain")
			}
			if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial report on failed temporal source")
			}
		}
	})
}
