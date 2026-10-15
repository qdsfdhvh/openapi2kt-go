package core_test

import (
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"testing"
)

func FuzzRecursiveCoverReferenceChains(f *testing.F) {
	b, err := os.ReadFile("../tests/fixtures/http/recursive-cover-reference-chains.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(b)
	f.Add(schemas(`{"Value":` + coverChainRoot("Value", "Value") + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`))
	f.Add(schemas(`{"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value") + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`))
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
				t.Fatal("nondeterministic recursive cover source graph")
			}
			if ea != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
				t.Fatal("failed recursive cover source returned artifacts/warnings")
			}
		}
	})
}
