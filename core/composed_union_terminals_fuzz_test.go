package core_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzComposedUnionTerminals(f *testing.F) {
	b, err := os.ReadFile("../tests/fixtures/http/composed-union-terminals.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Add(schemas(`{"Product":` + composedTerminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Product"}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/value","nullable":true}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 100000 {
			t.Skip()
		}
		for _, checked := range []bool{false, true} {
			r := request(b, "Dto")
			r.Options.Validation = checked
			a, err := core.GenerateWithDiagnostics(r)
			b, err2 := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(a, b) || fmt.Sprint(err) != fmt.Sprint(err2) {
				t.Fatal("nondeterministic product terminal")
			}
			if err != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("partial report on product failure")
			}
		}
	})
}
